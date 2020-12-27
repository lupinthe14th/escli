package main

// trimNexEqual は最初の=の次から末尾までの文字列を返す
func trimNextEqual(s string) string {
	i := 0
	for i = 0; i < len(s); i++ {
		if s[i] == '=' {
			break
		}
	}
	return s[i+1:]
}
