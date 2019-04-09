package permucombin

import "fmt"

// 设置一个时间来落地问题
var qHorseTime = map[string]float32{"q1": 1.0, "q2": 2.0, "q3": 3.0}
var tHorseTime = map[string]float32{"t1": 1.5, "t2": 2.5, "t3": 3.5}
var qHorses = []string{"q1", "q2", "q3"}
var tHorses = []string{"t1", "t2", "t3"}

func Permutate(Horses []string, result []string) {
	if len(Horses) == 0 {
		fmt.Println(result)
		Compare(result, qHorses)
		//returned
	}
	// 修正slice delete element
	for i := 0; i < len(Horses); i++ {
		var newResult = result
		newResult = append(newResult, Horses[i])
		var resetHorses = Horses
		resetHorses = append(resetHorses[:i], resetHorses[i+1:]...)
		Permutate(resetHorses, newResult)
	}
}

func Compare(t []string, q []string) {
	var tWinCnt int = 0
	for i := 0; i < len(t); i++ {
		tTime := tHorseTime[t[i]]
		qTime := qHorseTime[q[i]]
		fmt.Println(tTime, qTime)
		if tTime < qTime {
			tWinCnt++
		}
	}
	if tWinCnt > len(t)/2 {
		fmt.Println("t 赢了")
	} else {
		fmt.Println("q 赢了")
	}
}
