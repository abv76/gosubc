package arch

// Backend represents an interface a architecture
// specific code generator must have for the compiler to use
// for generating code.
type Backend interface {
	Add()
	Align()
	And()
	Bool()
	BrEq(n int)
	BrFalse(n int)
	BrGe(n int)
	BrGt(n int)
	BrLe(n int)
	BrLt(n int)
	BrNe(n int)
	BrTrue(n int)
	BrUge(n int)
	BrUgt(n int)
	BrUle(n int)
	BrUlt(n int)
	Call(s string)
	Calr()
	CalSwtch()
	Case(v, l int)
	Clear()
	Clear2()
	Data()
	Dec1ib()
	Dec1iw()
	Dec1pi(v int)
	Dec2ib()
	Dec2iw()
	Dec2pi(v int)
	Decgb(s string)
	Decgw(s string)
	Declb(a int)
	Declw(a int)
	Decpg(s string, v int)
	Decpl(a, v int)
	Decps(a, v int)
	Decsb(a int)
	Decsw(a int)
	Defb(v int)
	Defc(c int)
	Defl(v int)
	Defp(v int)
	Defw(v int)
	Div()
	Entry()
	Eq()
	Exit()
	Gbss(s string, z int)
	Ge()
	Gt()
	Inc1ib()
	Inc1iw()
	Inc1pi(v int)
	Inc2ib()
	Inc2iw()
	Inc2pi(v int)
	Incgb(s string)
	Incgw(s string)
	Inclb(a int)
	Inclw(a int)
	Incpg(s string, v int)
	Incpl(a, v int)
	Incps(a, v int)
	Incsb(a int)
	Incsw(a int)
	Indb()
	Indw()
	Initlw(v, a int)
	Or()
	Jump(n int)
	Lbss(s string, z int)
	Ldga(s string)
	Ldgb(s string)
	Ldgw(s string)
	Ldinc()
	Ldla(n int)
	Ldlab(id int)
	Ldlb(n int)
	Ldlw(n int)
	Ldsa(n int)
	Ldsb(n int)
	Ldsw(n int)
	LdSwtch(n int)
	Le()
	Lit(v int)
	Load2() bool
	LogNot()
	Lt()
	Mod()
	Mul()
	Ne()
	Neg()
	Not()
	Pop2()
	PopPtr()
	Postlude()
	Prelude()
	Public(s string)
	Push()
	PushLit(n int)
	Scale()
	Scale2()
	Scale2By(v int)
	ScaleBy(v int)
	Shl()
	Shr()
	Stack(n int)
	Storgb(s string)
	Storgw(s string)
	Storib()
	Storiw()
	Storlb(n int)
	Storlw(n int)
	Storsb(n int)
	Storsw(n int)
	Sub()
	Swap()
	Text()
	Uge()
	Ugt()
	Ule()
	Ult()
	Unscale()
	UnscaleBy(v int)
	Xor()
}
