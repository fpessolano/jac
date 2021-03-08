package jac

import "fmt"

func anything2String(x interface{}) string {
	return fmt.Sprintf("%v", x)
}
