package permucombin

import "fmt"

// 设置一个时间来落地问题
var qHorseTime = map[string]float32{"q1": 1.0, "q2": 2.0, "q3": 3.0}
var tHorseTime = map[string]float32{"t1": 1.5, "t2": 2.5, "t3": 3.5}
var qHorses = []string{"q1", "q2", "q3"}
var tHorses = []string{"t1", "t2", "t3"}

func Permutate(resetHorses []string, result []string) {
	if len(resetHorses) == 0 {
		fmt.Println(result)
		Compare(result, qHorses)
		return
	}

	for i := 0; i < len(resetHorses); i++ {
		newResult := result
		newResult = append(newResult, qHorses[i])
		resetHorses := resetHorses[i+1:]
		Permutate(resetHorses, newResult)
	}
}

func Compare(t []string, q []string) {
	var tWinCnt int
	for i := 0; i < len(t); i++ {
		if tHorseTime[t[i]] < qHorseTime[t[i]] {
			tWinCnt++
		}
	}
	if tWinCnt > len(t)/2 {
		fmt.Println("t 赢了")
	}
}
