package document

// NormalStep 是 Normalized Path 中的对象成员名或数组索引。
type NormalStep struct {
	name  string
	index int64
	isIdx bool
}

// NameStep 构造对象成员名路径段。
func NameStep(name string) NormalStep { return NormalStep{name: name} }

// IndexStep 构造数组索引路径段。
func IndexStep(i int64) NormalStep { return NormalStep{index: i, isIdx: true} }

// IsIndex 报告路径段是否为数组索引。
func (s NormalStep) IsIndex() bool { return s.isIdx }

// Name 返回对象成员名；数组索引路径段返回空字符串。
func (s NormalStep) Name() string { return s.name }

// Index 返回数组索引；对象成员名路径段返回零。
func (s NormalStep) Index() int64 { return s.index }
