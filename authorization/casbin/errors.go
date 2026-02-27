package casbin

type UnknownPolicyTypeError struct {
	PolicyType string
}

func (err UnknownPolicyTypeError) Error() string {
	return "unknown policy type: " + err.PolicyType
}
