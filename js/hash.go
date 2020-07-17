package js

// uses github.com/tdewolff/hasher
//go:generate hasher -type=Hash -file=hash.go

// Hash defines perfect hashes for a predefined list of strings
type Hash uint32

// Identifiers for the hashes associated with the text in the comments.
const (
	As         Hash = 0x2    // as
	Async      Hash = 0x5    // async
	Await      Hash = 0x4305 // await
	Break      Hash = 0x5c05 // break
	Case       Hash = 0x404  // case
	Catch      Hash = 0x6905 // catch
	Class      Hash = 0xd605 // class
	Const      Hash = 0x6105 // const
	Continue   Hash = 0x6e08 // continue
	Debugger   Hash = 0x2b08 // debugger
	Default    Hash = 0xca07 // default
	Delete     Hash = 0xde06 // delete
	Do         Hash = 0x7602 // do
	Else       Hash = 0x1004 // else
	Enum       Hash = 0x1304 // enum
	Export     Hash = 0x2006 // export
	Extends    Hash = 0x5007 // extends
	False      Hash = 0x4c05 // false
	Finally    Hash = 0x8807 // finally
	For        Hash = 0xa803 // for
	From       Hash = 0x7804 // from
	Function   Hash = 0x7c08 // function
	Get        Hash = 0x1b03 // get
	If         Hash = 0x8702 // if
	Implements Hash = 0x8f0a // implements
	Import     Hash = 0x9906 // import
	In         Hash = 0x7202 // in
	Instanceof Hash = 0x9f0a // instanceof
	Interface  Hash = 0xab09 // interface
	Let        Hash = 0xe003 // let
	Meta       Hash = 0x1604 // meta
	New        Hash = 0x3703 // new
	Null       Hash = 0x8304 // null
	Of         Hash = 0x4b02 // of
	Package    Hash = 0xb407 // package
	Private    Hash = 0xbb07 // private
	Protected  Hash = 0xc209 // protected
	Public     Hash = 0xd106 // public
	Return     Hash = 0x3206 // return
	Set        Hash = 0x603  // set
	Static     Hash = 0x6406 // static
	Super      Hash = 0x3e05 // super
	Switch     Hash = 0x5606 // switch
	Target     Hash = 0x1806 // target
	This       Hash = 0x3b04 // this
	Throw      Hash = 0x805  // throw
	True       Hash = 0x1d04 // true
	Try        Hash = 0x2503 // try
	Typeof     Hash = 0x4706 // typeof
	Void       Hash = 0xdb04 // void
	While      Hash = 0xc05  // while
	With       Hash = 0x3904 // with
	Yield      Hash = 0x2705 // yield
)

var HashMap = map[string]Hash{
	"as":         As,
	"async":      Async,
	"await":      Await,
	"break":      Break,
	"case":       Case,
	"catch":      Catch,
	"class":      Class,
	"const":      Const,
	"continue":   Continue,
	"debugger":   Debugger,
	"default":    Default,
	"delete":     Delete,
	"do":         Do,
	"else":       Else,
	"enum":       Enum,
	"export":     Export,
	"extends":    Extends,
	"false":      False,
	"finally":    Finally,
	"for":        For,
	"from":       From,
	"function":   Function,
	"get":        Get,
	"if":         If,
	"implements": Implements,
	"import":     Import,
	"in":         In,
	"instanceof": Instanceof,
	"interface":  Interface,
	"let":        Let,
	"meta":       Meta,
	"new":        New,
	"null":       Null,
	"of":         Of,
	"package":    Package,
	"private":    Private,
	"protected":  Protected,
	"public":     Public,
	"return":     Return,
	"set":        Set,
	"static":     Static,
	"super":      Super,
	"switch":     Switch,
	"target":     Target,
	"this":       This,
	"throw":      Throw,
	"true":       True,
	"try":        Try,
	"typeof":     Typeof,
	"void":       Void,
	"while":      While,
	"with":       With,
	"yield":      Yield,
}

// String returns the text associated with the hash.
func (i Hash) String() string {
	return string(i.Bytes())
}

// Bytes returns the text associated with the hash.
func (i Hash) Bytes() []byte {
	start := uint32(i >> 8)
	n := uint32(i & 0xff)
	if start+n > uint32(len(_Hash_text)) {
		return []byte{}
	}
	return _Hash_text[start : start+n]
}

// ToHash returns a hash Hash for a given []byte. Hash is a uint32 that is associated with the text in []byte. It returns zero if no match found.
func ToHash(s []byte) Hash {
	//if 3 < len(s) {
	//	return HashMap[string(s)]
	//}
	h := uint32(_Hash_hash0)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	if i := _Hash_table[h&uint32(len(_Hash_table)-1)]; int(i&0xff) == len(s) {
		t := _Hash_text[i>>8 : i>>8+i&0xff]
		for i := 0; i < len(s); i++ {
			if t[i] != s[i] {
				goto NEXT
			}
		}
		return i
	}
NEXT:
	if i := _Hash_table[(h>>16)&uint32(len(_Hash_table)-1)]; int(i&0xff) == len(s) {
		t := _Hash_text[i>>8 : i>>8+i&0xff]
		for i := 0; i < len(s); i++ {
			if t[i] != s[i] {
				return 0
			}
		}
		return i
	}
	return 0
}

const _Hash_hash0 = 0xe5a143d2
const _Hash_maxLen = 10

var _Hash_text = []byte("" +
	"asyncasethrowhilelsenumetargetruexportryieldebuggereturnewit" +
	"hisuperawaitypeofalsextendswitchbreakconstaticatchcontinuedo" +
	"fromfunctionullifinallyimplementsimportinstanceoforinterface" +
	"packageprivateprotectedefaultpubliclassvoidelete")

var _Hash_table = [1 << 6]Hash{
	0x0:  0x8f0a, // implements
	0x1:  0x2b08, // debugger
	0x2:  0xd605, // class
	0x3:  0x1304, // enum
	0x4:  0x4305, // await
	0x5:  0x8702, // if
	0x6:  0x7c08, // function
	0x7:  0x1004, // else
	0x8:  0x603,  // set
	0xa:  0xe003, // let
	0xb:  0xd106, // public
	0xc:  0x9906, // import
	0xd:  0x3703, // new
	0xe:  0x2,    // as
	0xf:  0x6105, // const
	0x11: 0x3e05, // super
	0x13: 0x1806, // target
	0x14: 0xb407, // package
	0x15: 0x6406, // static
	0x16: 0x5007, // extends
	0x17: 0x7602, // do
	0x19: 0x8304, // null
	0x1b: 0x1d04, // true
	0x1c: 0x3206, // return
	0x1d: 0x7202, // in
	0x1e: 0x404,  // case
	0x1f: 0x2705, // yield
	0x20: 0x9f0a, // instanceof
	0x21: 0xa803, // for
	0x22: 0x3b04, // this
	0x23: 0x4706, // typeof
	0x25: 0xab09, // interface
	0x26: 0x5,    // async
	0x27: 0x4b02, // of
	0x28: 0xc209, // protected
	0x29: 0x805,  // throw
	0x2a: 0x6905, // catch
	0x2d: 0xde06, // delete
	0x2f: 0xca07, // default
	0x30: 0x2006, // export
	0x31: 0xbb07, // private
	0x33: 0x2503, // try
	0x34: 0x8807, // finally
	0x35: 0x3904, // with
	0x36: 0x7804, // from
	0x37: 0xc05,  // while
	0x38: 0x4c05, // false
	0x39: 0x5c05, // break
	0x3a: 0x5606, // switch
	0x3b: 0x6e08, // continue
	0x3c: 0xdb04, // void
	0x3e: 0x1b03, // get
	0x3f: 0x1604, // meta
}
