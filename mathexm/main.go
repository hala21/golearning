package main

import (
	"golearning/mathexm/iterdemo"
	"golearning/mathexm/remainder"
)

func main() {
	wheats := iterdemo.GetNumberOfWheatItera(63)
	println(wheats)
	remainder := remainder.Remainder
	println(remainder)
}
