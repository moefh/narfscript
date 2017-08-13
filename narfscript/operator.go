package narfscript

type operatorAssoc int32

const minOperatorPrec int32 = -(int32(^uint32(0) >> 1)) - 1

const (
	operatorAssocLeft operatorAssoc = iota
	operatorAssocRight
	operatorAssocPrefix
)

type bleepOperator struct {
	ident string
	prec  int32
	assoc operatorAssoc
}
