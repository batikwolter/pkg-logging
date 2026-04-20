package observe

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ErrorKindBusiness = "business"
	ErrorKindInternal = "internal"
)

type ErrorClassification struct {
	Code    string
	Kind    string
	Outcome string
}

func GRPCCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	st, ok := status.FromError(err)
	if ok {
		return st.Code()
	}
	return codes.Unknown
}

func IsBusinessError(err error) bool {
	return IsBusinessCode(GRPCCode(err))
}

func IsBusinessCode(code codes.Code) bool {
	switch code {
	case codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.FailedPrecondition,
		codes.OutOfRange,
		codes.Unauthenticated,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.Canceled:
		return true
	default:
		return false
	}
}

func ClassifyError(err error) ErrorClassification {
	if err == nil {
		return ErrorClassification{
			Code:    codes.OK.String(),
			Outcome: "success",
		}
	}

	code := GRPCCode(err)
	kind := ErrorKindInternal
	if IsBusinessCode(code) {
		kind = ErrorKindBusiness
	}

	return ErrorClassification{
		Code:    code.String(),
		Kind:    kind,
		Outcome: "error",
	}
}

func ErrorFields(err error) []Field {
	if err == nil {
		return nil
	}
	cls := ClassifyError(err)
	message := status.Convert(err).Message()
	if message == "" {
		message = err.Error()
	}

	return []Field{
		String("error.message", message),
		String("error.code", cls.Code),
		String("error.kind", cls.Kind),
	}
}
