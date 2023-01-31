package acl

import "hash"

// EnterpriseMeta stub
type EnterpriseMetadata interface {
	ToEnterprisePolicyMeta() *EnterprisePolicyMeta
	EstimateSize() int
	AddToHash(_ hash.Hash, _ bool)
	PartitionOrDefault() string
	PartitionOrEmpty() string
	InDefaultPartition() bool
	NamespaceOrDefault() string
	NamespaceOrEmpty() string
	InDefaultNamespace() bool
	Merge(_ EnterpriseMetadata)
	MergeNoWildcard(_ EnterpriseMetadata)
	Normalize()
	NormalizePartition()
	NormalizeNamespace()
	Matches(_ EnterpriseMetadata) bool
	IsSame(_ EnterpriseMetadata) bool
	LessThan(_ EnterpriseMetadata) bool
	WithWildcardNamespace() EnterpriseMetadata
	UnsetPartition()
	OverridePartition(_ string)
	FillAuthzContext(_ *AuthorizerContext)
}
