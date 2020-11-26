package options

func boolPtr(flag bool) *bool {
	return &flag
}

func strPtr(str string) *string {
	return &str
}

func intPtr(num int) *int {
	return &num
}
