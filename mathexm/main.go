package main

import (
	"fmt"
	"golearning/mathexm/iterdemo"
	"golearning/mathexm/mathinduction"
	"golearning/mathexm/remainder"
	"time"
)

func main() {
	//
	wheats := iterdemo.GetNumberOfWheatItera(63)
	println(wheats)
	// 3.取模
	var remaind = remainder.Remainder(7)
	//println(strconv.Itoa(int(remainder)))
	fmt.Printf("The result is: %v\n", remaind)
	// 4.递归
	t1 := time.Now().UnixNano()
	detal := iterdemo.GetSquareRoot(100, 0.000000001, 100)
	t2 := time.Now().UnixNano()
	fmt.Println(t2 - t1)
	fmt.Println(detal)
	// 4.数学归纳法
	var pro = &mathinduction.Result{}
	result := mathinduction.Prove(6, pro)
	fmt.Println(result)
	// 5. 递归 泛化数学归纳
	mathinduction.GetRewardsRecursion1(20, []int{})
	// 6. 分而治之，从归并排序到MapReduce，跑了才知道程序是这样运行的，递归是这样做的，学到了
	mergesort := mathinduction.MergeSort([]int{243, 343, 6546, 32, 432, 42343, 423})
	fmt.Println(mergesort)

}
