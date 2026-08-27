package domain

import "errors"

var (
	ErrNotFound         = errors.New("资源不存在")
	ErrVersionConflict  = errors.New("版本冲突")
	ErrInvalidState     = errors.New("当前状态不允许此操作")
	ErrIdentityConflict = errors.New("复核人不能是修订提交者")
	ErrBlockingFindings = errors.New("仍有未关闭的阻断问题")
	ErrStaleValidation  = errors.New("校验依据已过期")
	ErrFrozen           = errors.New("项目已经冻结")
	ErrInvalidInput     = errors.New("输入无效")
)
