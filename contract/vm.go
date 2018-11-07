package contract

type Context struct {
}

type Result struct {
	err error
}

type CallInfo struct {
	method string
	args   [][]byte
}
