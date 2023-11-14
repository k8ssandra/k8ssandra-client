package util

import "math/rand"

func RandomKubeCompatibleText(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset)-1)]
	}
	gened := string(b)

	return gened
}
