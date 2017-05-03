//go:generate stringer -type op
package asm

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"unicode"
)

// Assemble assembles an operation.
func Assemble(arch, os_, input string, output io.Writer, src []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
			if _, ok := err.(runtime.Error); ok {
				panic(err)
			}
		}
	}()
	prog := newprog(arch, os_)
	switch arch {
	case "amd64":
		x86as(prog, input, src)
	default:
		return fmt.Errorf("unsupported arch %q", arch)
	}

	w := bufio.NewWriter(output)
	switch os_ {
	case "linux":
		genelf(w, prog)
	default:
		return fmt.Errorf("unsupported os %q", os_)
	}

	return w.Flush()
}

// generic op for all architectures.
const (
	opNOP op = iota
	opSTRZ
	opQUAD
	opLONG
	opSHORT
	opBYTE
	opBYTES
)

// addressing mode for all architectures.
const (
	aNONE = iota
	aREG
	aVAR
	aPTR
	aMEM
	aINT
	aSTR
	aNOTE
	aSECT
)

// a variable type this will decide which section
// it is placed into and the relocations applied to it
const (
	sNONE = iota
	sLABEL
	sBSS
	sUND
)

// relocation types for all architectures.
const (
	lN = iota
	lS
	lPC
	lV
)

// section type
const (
	stNONE = iota
	stNOTE
	stPROGBITS
	stNOBITS
	stSYMTAB
	stSTRTAB
)

// op represents an symbolic opcode.
type op int

// as contains the generic assembler data structure
// shared by all architectures.
type as struct {
	*prog
	sect   *section
	file   string
	line   string
	lineno int64
}

// relocation represents a relocation.
type relocation struct {
	*section
	*inst
	off   int64
	pc    int
	isize int

	reltyp  int
	relname string
	rel     int64
}

// addr represents an argument for the instruction.
type addr struct {
	typ   int
	reg   byte
	ival  int64
	sval  string
	deref bool
}

// inst represents an instruction.
type inst struct {
	op   op
	addr [4]addr
	code []byte
}

// section represents one section.
type section struct {
	name       string
	flags      string
	typ        int
	inst       []*inst
	syms       []*sym
	vars       []*sym
	labels     []*sym
	blocks     []*sym
	strings    []span
	relocs     []*relocation
	size       int64
	blockalign int64
	blocksize  int64
	pc         int
}

// span represents a range.
type span struct {
	off  int64
	size int64
	pc   int
}

// prog contains all the information generated by the assembler.
// This is used to emit an object file.
type prog struct {
	arch   string
	os     string
	endian binary.ByteOrder
	text   *section
	data   *section
	bss    *section
	sects  []*section
	syms   map[string]*sym
	osyms  []*sym
	usyms  []*sym
	relocs []*relocation
}

// sym represents a symbol.
type sym struct {
	sect      *section
	typ       int
	name      string
	size      int64
	off       int64
	pc        int
	allocated bool
	exported  bool
}

// newprog creates an empty prog
// with the architecture information.
func newprog(arch, os_ string) *prog {
	return &prog{
		arch:   arch,
		os:     os_,
		endian: binary.LittleEndian,
		syms:   make(map[string]*sym),
		text:   newsection(".text", "ax", stPROGBITS),
		data:   newsection(".data", "wa", stPROGBITS),
		bss:    newsection(".bss", "wa", stNOBITS),
	}
}

// newsection creates a new section.
func newsection(name, flags string, typ int) *section {
	return &section{
		name:  name,
		flags: flags,
		typ:   typ,
	}
}

// errorf outputs an error string by the assembler.
func (as *as) errorf(format string, args ...interface{}) {
	var pos string
	if as.file != "" {
		pos = fmt.Sprintf("%s:%d", as.file, as.lineno)
	} else {
		pos = fmt.Sprint(as.lineno)
	}
	text := fmt.Sprintf(format, args...)
	if as.line != "" {
		errf("%s: error: %q\n%*s%s", pos, as.line, len(pos)+9, "", text)
	} else {
		errf("%s: error: %s", pos, text)
	}
}

// code emits a buffer of machine code.
func (as *as) code(v ...interface{}) []byte {
	var code []byte
	buf := new(bytes.Buffer)
	for i := range v {
		buf.Reset()
		switch v := v[i].(type) {
		case nil:
			continue
		case byte:
			code = append(code, byte(v))
		case int:
			code = append(code, byte(v))
		case uint16, uint32, uint64:
			binary.Write(buf, as.endian, v)
			code = append(code, buf.Bytes()...)
		default:
			as.errorf("unknown emit type %T", v)
		}
	}
	return code
}

// emit emits code for an instruction.
func (as *as) emit(op op, addr [4]addr, v ...interface{}) {
	code := as.code(v...)
	s := as.sect
	s.inst = append(s.inst, &inst{op: op, addr: addr, code: code})
	s.size += int64(len(code))
}

// gsym adds a variable to the global symbol table
// and add it to the section it corresponds to.
// if the variable does exist, it will just return that.
func (as *as) gsym(name string, s *section) *sym {
	p := as.syms[name]
	if p != nil {
		return p
	}

	s.syms = append(s.syms, &sym{})
	p = s.syms[len(s.syms)-1]
	p.sect = s
	p.name = name
	p.allocated = true
	as.syms[name] = p
	as.osyms = append(as.osyms, p)

	return p
}

// fsym looks up a variable from the global symbol table.
// If it does exist, it will add it to the undefined symbol
// table for use by the linker.
func (as *as) fsym(typ int, name string) *sym {
	if (typ != aPTR && typ != aVAR) || name == "" {
		return nil
	}
	s := as.syms[name]
	if s == nil {
		as.usyms = append(as.usyms, &sym{
			typ:      sUND,
			name:     name,
			exported: true,
		})
		s = as.usyms[len(as.usyms)-1]
		as.syms[name] = s
		as.osyms = append(as.osyms, s)
	}
	return s
}

// addglobal marks a global variable as exported.
// If the variable does not exist, it will create
// one of no type.
func (as *as) addglobal(name string) {
	p := as.gsym(name, as.sect)
	p.exported = true
}

// addbss adds a bss variable.
func (as *as) addbss(name string, size int64, allocated bool) {
	p := as.gsym(name, as.bss)
	if p.typ == sNONE {
		p.typ = sBSS
		p.size = size
		p.allocated = allocated
		if allocated {
			p.sect.blocksize += size
		}
		as.bss.blocks = append(as.bss.blocks, p)
		return
	}
	as.errorf("%q already declared", name)
}

// addlabel adds a label.
func (as *as) addlabel(name string, off int64, pc int) {
	p := as.gsym(name, as.sect)
	if p.typ == sLABEL {
		if p.off != off {
			goto fail
		}
		return
	}
	if p.typ == sNONE {
		p.typ = sLABEL
		p.off = off
		p.pc = pc
		as.sect.labels = append(as.sect.labels, p)
		return
	}

fail:
	as.errorf("%q already declared", name)
}

// addrel adds a relocation.
func (as *as) addrel(op op, addr [4]addr) {
	s := as.sect
	s.inst = append(s.inst, &inst{op: op, addr: addr})
	i := s.inst[len(s.inst)-1]

	as.relocs = append(as.relocs, &relocation{section: s, inst: i, off: s.size, pc: s.pc})
	s.relocs = append(s.relocs, as.relocs[len(as.relocs)-1])
}

// addsect adds a section.
func (as *as) addsect(name, flags, typ string) {
	switch name {
	case ".text", ".data", ".rela.text", ".rela.data",
		".bss", ".shstrtab", ".strtab", ".symtab":
		as.errorf("can't define a pre-defined section %q", name)
	case "":
		as.errorf("no section name specified")
	}

	var xtyp int
	switch typ {
	case "progbits":
		xtyp = stPROGBITS
	case "nobits":
		xtyp = stNOBITS
	case "note":
		xtyp = stNOTE
	default:
		as.errorf("unknown section type %q", typ)
	}

	for _, s := range as.sects {
		if s.name == name {
			as.sect = s
			return
		}
	}
	as.sects = append(as.sects, newsection(name, flags, xtyp))
	as.sect = as.sects[len(as.sects)-1]
}

// alignpc aligns current pc to value.
func (as *as) alignpc(align int, value uint8) {
	size := (align - (as.sect.pc % align)) % align
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = value
	}
	as.sect.bytes(buf)
}

// strz appends a nul-terminated string to the instruction stream.
func (s *section) strz(str string) {
	s.strings = append(s.strings, span{
		off:  s.size,
		size: int64(len(str) + 1),
		pc:   s.pc,
	})

	s.size += int64(len(str)) + 1
	s.inst = append(s.inst, &inst{
		op:   opSTRZ,
		code: append([]byte(str), 0),
	})
}

// bytes emits a byte buffer to the instruction stream.
// the buffer is not copied but used directly.
func (s *section) bytes(buf []byte) {
	s.size += int64(len(buf))
	s.inst = append(s.inst, &inst{
		op:   opBYTES,
		code: buf,
	})
}

// isIdent returns if a string is an identifier.
func isIdent(s string) bool {
	for i, r := range s {
		isAlpha := r == '_' || unicode.IsLetter(r)
		if i == 0 && !isAlpha {
			return false
		}
		if i > 0 && !isAlpha && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// align2 aligns to the next power of 2.
func align2(x int64) int64 {
	y := int64(1)
	for y < x {
		y <<= 1
	}
	return y
}

func errf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}
