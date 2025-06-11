package models

type MultiError struct {
	errs []error
}

func NewMultiError(errs []error) *MultiError {
	return &MultiError{errs: errs}
}

func (e *MultiError) Error() string {
	var str string
	errsLen := len(e.errs)
	for i, err := range e.errs {
		str += err.Error()
		if i < errsLen-1 {
			str += "\n"
		}
	}
	return str
}

func (e *MultiError) Errors() []error {
	return e.errs
}

type RepoErr struct {
	Source string
	err    error
}

func NewRepoErr(source string, err error) *RepoErr {
	return &RepoErr{
		Source: source,
		err:    err,
	}
}

func (e *RepoErr) Error() string {
	return e.err.Error()
}

func (e *RepoErr) Unwrap() error {
	return e.err
}

type RepoModuleErr struct {
	Source  string
	Channel string
	err     error
}

func NewRepoModuleErr(source, channel string, err error) *RepoModuleErr {
	return &RepoModuleErr{
		Source:  source,
		Channel: channel,
		err:     err,
	}
}

func (e *RepoModuleErr) Error() string {
	return e.err.Error()
}

func (e *RepoModuleErr) Unwrap() error {
	return e.err
}
