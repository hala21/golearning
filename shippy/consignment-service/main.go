package main

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net"
	pb "shippy/consignment-service/proto/consignment"
)

type IRepository interface{
	Create(consignment *pb.Consignment) (*pb.Consignment,error) //chunfanghuowu
	GetAll() []*pb.Consignment
}

type Repository struct {
	consignments []*pb.Consignment
}

func (repo *Repository)Create(consignment *pb.Consignment)(*pb.Consignment, error){
	repo.consignments = append(repo.consignments, consignment)
	return  consignment ,nil
}

func (repo *Repository)GetAll() []*pb.Consignment{
	return repo.consignments
}

type service struct {
	repo Repository
}

func (s *service)CreateConsignment(cxt context.Context, req *pb.Consignment)(*pb.Response, error){
	consignment, err := s.repo.Create(req)
	if err != nil {
		return nil, err
	}

	return &pb.Response{Created: true , Consignment: consignment}, nil
}

func (s *service)GetConsignments(cxt context.Context, req *pb.GetRequest)(*pb.Response, error){
	consignments := s.repo.GetAll()
	return &pb.Response{Consignments: consignments}, nil
}

func main() {

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}

	s := grpc.NewServer()

	repo := Repository{}

	pb.RegisterShippingServiceServer(s, &service{repo})

	if err := s.Serve(lis); err != nil{
		log.Fatalf("server error : %v", err)
	}

}