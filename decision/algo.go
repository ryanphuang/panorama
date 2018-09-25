package decision

import (
	pb "panorama/build/gen"
)

type InferenceAlgo interface {
	InferPano(panorama *pb.Panorama, workbook map[string]*pb.Inference) *pb.Inference
	InferView(view *pb.View) *pb.Inference
}
