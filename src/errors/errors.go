package errors

import "strconv"

type GoRefactorError struct{
	
	Code int
	ErrorType string
	Message string
}

func (err *GoRefactorError) String() string{
	return err.ErrorType + ": " + err.Message;
}

func IdentifierNotFoundError(filename string, line int, column int) *GoRefactorError{
	return &GoRefactorError{1,"position error", "no entity found at position "+ filename + " " + strconv.Itoa(line) + ":" + strconv.Itoa(column)};
}

func IdentifierAlreadyExistsError(name string) *GoRefactorError{
	return &GoRefactorError{5,"identifier already exists error", "identifier "+ name + " already exists in current context."};
}

func UnrenamableIdentifierError(name string, reason string) *GoRefactorError{
		
	return &GoRefactorError{2,"unrenamable identifier error", "identifier " + name +" can not be renamed. " + reason};
}

func ParsingError(packageName string) *GoRefactorError{
	
	return &GoRefactorError{3,"parsing error", "an error occured while parsing package " + packageName + "."};
}

func ArgumentError(parameterName string,reason string) *GoRefactorError{
	
	return &GoRefactorError{4,"argument error", "parameter "+ parameterName +" has invalid value. " + reason};
}

func PrinterError(message string) *GoRefactorError{
	
	return &GoRefactorError{6,"printer error", message};
}