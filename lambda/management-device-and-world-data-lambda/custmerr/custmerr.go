package custmerr

//本来はerrorsパッケージのみで対処したいところだが､呼び出し元のエラーハンドリンで識別しにくいので定義する

// データが存在しないなどの論理的なエラーを表す
type LogicalErr struct {
	Err error
}

func (e *LogicalErr) Error() string {
	return e.Err.Error()
}

func (e *LogicalErr) Unwrap() error {
	return e.Err
}

// トランザクショが貼れなかったなどの技術エラーを表す
type TechnicalErr struct {
	Err error
}

func (e *TechnicalErr) Error() string {
	return e.Err.Error()
}

func (e *TechnicalErr) Unwrap() error {
	return e.Err
}
