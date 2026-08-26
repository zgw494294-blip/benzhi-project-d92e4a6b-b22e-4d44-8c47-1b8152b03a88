package ledger

import "fmt"

type IntegrityError struct {
	Sequence uint64
	Reason   string
}

func (e *IntegrityError) Error() string {
	if e.Sequence == 0 {
		return "账本完整性错误: " + e.Reason
	}
	return fmt.Sprintf("账本完整性错误（序号 %d）: %s", e.Sequence, e.Reason)
}
