package main 
import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func main(){
	
	// generating the random api secret key 

	bytes:=make([]byte,32)
	if _,err:=rand.Read(bytes); err!=nil{
		panic(err)
	}
	fmt.Println("Generated API Key: ",hex.EncodeToString(bytes))
}