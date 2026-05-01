package errors

// import "github.com/lib/pq"

// func NewPQForeignError() error {
// 	return &pq.Error{
// 		Code:     "23503",
// 		Severity: "ERROR",
// 		Message:  "insert or update on table \"orders\" violates foreign key constraint \"orders_userid_fkey\"",
// 		Detail:   "Key (userId)=(999) is not present in table \"users\".",
// 	}
// }
// func NewPQUniqueError() error {
// 	return &pq.Error{
// 		Code:     "23505",
// 		Severity: "ERROR",
// 		Message:  "duplicate key value violates unique constraint",
// 		Detail:   "Key (username)=(duplicate_user) already exists.",
// 	}
// }
// func NewPQCheckViolationError() error {
// 	return &pq.Error{
// 		Code:     "23514",
// 		Severity: "ERROR",
// 		Message:  "new row for relation violates check constraint",
// 		Detail:   "Failing row contains (x, -y).",
// 	}
// }

// func NewPQNotNullError() error {
// 	return &pq.Error{
// 		Code:     "23502",
// 		Severity: "ERROR",
// 		Message:  "null value in column violates not-null constraint",
// 		Detail:   "Failing row contains (x, null, YYY-MM-DD HH:mm:SS).",
// 	}
// }

// func NewPQSyntaxError() error {
// 	return &pq.Error{
// 		Code:     "22P02",
// 		Severity: "ERROR",
// 		Message:  "invalid input syntax for type",
// 		Detail:   "",
// 	}
// }
