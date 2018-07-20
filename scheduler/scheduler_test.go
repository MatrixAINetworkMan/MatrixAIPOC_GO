package scheduler
import (
	"testing"
	"fmt"
	//"time"
	"time"
)

func TestSkeduler(t *testing.T)  {
	a :=new(Scheduler)
	fmt.Printf("你好，世界\n")
	a.Start()

}


func set(buffer []int,a []int){
	temp :=make([]int,5,10)

	//buffer=temp
	copy(buffer,temp)
}

func get(buffer []int)(a []int){
	temp :=[]int{6, 7, 8, 9, 10}
	temp=buffer
	fmt.Println("values:",buffer)
	buffer[0]=100
	return  temp

}
func TestGetNodeList(t *testing.T)  {
	a :=new(Scheduler)
	go a.Start()
	time.Sleep(1000000000)
	nodelist, _:= a.Getmainnodelist()
	fmt.Print("nodelist len",len(nodelist))

}



func TestSetNodeListNotify(t *testing.T)  {
	ch:=make(chan int,1)

	select {
	case ch<-0:
		i:=<-ch
		fmt.Println("values:",i)
	case ch<-1:
		i:=<-ch
		fmt.Println("values:",i)
    }
}


func TestofflineListmortgagedeal(t *testing.T)  {

}