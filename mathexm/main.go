package main

import (
	"fmt"
	"golearning/mathexm/iterdemo"
	"golearning/mathexm/mathinduction"
	"golearning/mathexm/remainder"
	"time"
)

func main() {
	wheats := iterdemo.GetNumberOfWheatItera(63)
	println(wheats)
	var remaind = remainder.Remainder(7)
	//println(strconv.Itoa(int(remainder)))
	fmt.Printf("The result is: %v\n", remaind)
	t1 := time.Now().UnixNano()
	detal := iterdemo.GetSquareRoot(100, 0.000000001, 100)
	t2 := time.Now().UnixNano()
	fmt.Println(t2 - t1)
	fmt.Println(detal)
	var pro = &mathinduction.Result{}
	var result = pro.Prove(3)
	fmt.Println(result)

}
