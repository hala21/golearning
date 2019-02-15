package main

import (
	"encoding/json"
	"errors"
	"golang.org/x/net/context"
	pb "golearning/shippy/consignment-service/proto/consignment"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
)

func parseFile(filename string) (*pb.Consignment, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var consignment *pb.Consignment

	err = json.Unmarshal(data, &consignment)
	if err != nil {
		return nil, errors.New("deal with consignment file context error ")
	}
	return consignment, nil
}

func main() {

	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("grpc dial connect error : %v", err)
	}

	client := pb.NewShippingServiceClient(conn)

	//consignmentFile := "consignment.json"
	//if len(os.Args) >1 {
	//	consignmentFile = os.Args[1]
	//}

	//consignment, err := parseFile(consignmentFile)
	cxt := context.Background()

	//resp, err := client.CreateConsignment(cxt, consignment)
	//if err != nil {
	//	log.Fatalf("create consignment error: %v", err)
	//}
	//// 新货物是否托运成功
	//log.Printf("created: %t", resp.Created)

	respAll, err := client.GetConsignments(cxt, &pb.GetRequest{})
	if err != nil {
		log.Printf("get consigment falut: %v", err)
	}

	for _, con := range respAll.Consignments {
		log.Printf("%+v", con)
	}

}
