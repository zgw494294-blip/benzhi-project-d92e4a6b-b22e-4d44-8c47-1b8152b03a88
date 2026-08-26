package certification

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

func validateText(label, value string, minimum, maximum int) error {
	length := len([]rune(strings.TrimSpace(value)))
	if length < minimum || length > maximum {
		return domainError(CodeValidation, "%s 长度必须在 %d 到 %d 个字符之间", label, minimum, maximum)
	}
	return nil
}

func validateChineseReason(label, value string) error {
	if err := validateText(label, value, 2, 1000); err != nil {
		return err
	}
	for _, character := range strings.TrimSpace(value) {
		if unicode.Is(unicode.Han, character) {
			return nil
		}
	}
	return domainError(CodeValidation, "%s 必须包含中文说明", label)
}

func validateCaseFields(command CreateCaseCommand) error {
	fields := []struct {
		label string
		value string
		max   int
	}{
		{"工作台编号", command.CabinetCode, 80},
		{"安装位置", command.Location, 200},
		{"设备级别", command.CabinetClass, 80},
	}
	for _, field := range fields {
		if err := validateText(field.label, field.value, 1, field.max); err != nil {
			return err
		}
	}
	return nil
}

func validateEvidence(value string) error {
	if err := validateText("证据摘要", value, 8, 256); err != nil {
		return err
	}
	for _, character := range value {
		if character < 0x20 {
			return domainError(CodeValidation, "证据摘要不能包含控制字符")
		}
	}
	return nil
}

func validatePerson(label, value string) error {
	return validateText(label, value, 1, 100)
}

func validateDueDate(dueAt, now time.Time) error {
	if dueAt.IsZero() || !dueAt.After(now) {
		return domainError(CodeValidation, "偏差期限必须晚于当前时间")
	}
	if dueAt.After(now.AddDate(1, 0, 0)) {
		return domainError(CodeValidation, "偏差期限不能超过一年")
	}
	return nil
}

func validateExpectedVersion(expected int64) error {
	if expected <= 0 {
		return domainError(CodeValidation, "expectedVersion 必须为正整数")
	}
	if expected > 1<<53-1 {
		return domainError(CodeValidation, fmt.Sprintf("expectedVersion %d 超出安全范围", expected))
	}
	return nil
}
