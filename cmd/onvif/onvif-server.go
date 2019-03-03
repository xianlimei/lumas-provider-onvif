package main

import (
  "fmt"
  "flag"
  "errors"
  "log"
  "net"
  "context"
  "time"
  "google.golang.org/grpc"

  . "github.com/lumas-ai/lumas-provider-onvif"
  api "github.com/lumas-ai/lumas-core/protos/golang/provider"
)

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	iface      = flag.String("host", "0.0.0.0", "The interface to listen on")
	port       = flag.Int("port", 5390, "The server port")
)

type CameraServer struct { }

var (
  cameras map[string]*Camera
)

func (s *CameraServer) StreamRTP(config *api.RTPConfig, stream api.Camera_StreamRTPServer) error {
  camera := &Camera{Config: config}
  cameraID, err := camera.GenerateCameraID()
  defer camera.Close()
  if err != nil {
    return err
  }

  if cameras == nil {
    cameras = make(map[string]*Camera)
  }
  cameras[cameraID] = camera

  asdp := make(chan string)
  vsdp := make(chan string)
  errors := make(chan error)

  go func() {
    err := camera.StartRTPStream(vsdp, asdp)
    if err != nil {
      errors <- err
    }
  }()

  videoOutputSDP := <-vsdp
  audioOutputSDP := <-asdp

  //Send the first response with the SDP information
  sdp := api.SDP{
    Audio: audioOutputSDP,
    Video: videoOutputSDP,
  }
  r := api.StreamInfo{
    Sdp: &sdp,
  }
  stream.Send(&r)

  //Send a status update every second
  for {
    //Check for errors and continue if there
    //are none
    select {
    case err := <-errors:
      return err
    default:
      //continue on with loop
    }

    //the StopRTPStream call will remove the camera
    //from the cameras map, so we can check for its non-existence
    //to know if we need to stop looping
    if cameras[cameraID] == nil {
      break
    }

    r := api.StreamInfo{
      SentFrames: int64(camera.SentFrames),
      DroppedFrames: int64(camera.DroppedFrames),
    }
    if err := stream.Send(&r); err != nil {
      break
    }

    time.Sleep(1 * time.Second)
  }

  return nil
}

func (s *CameraServer) Snapshot(ctx context.Context, config *api.CameraConfig) (*api.Image, error) {
  return new(api.Image), nil
}

func (s *CameraServer) StopRTPStream(context context.Context, config *api.RTPConfig) (*api.Result, error) {
  cam := &Camera{Config: config}
  cameraID, _ := cam.GenerateCameraID()
  camera := cameras[cameraID]

  if camera == nil {
    r := api.Result{
      Successful: false,
      ErrorKind: "StreamNotFound",
      Message: "Camera stream not found",
    }
    return &r, errors.New("Camera stream not found")
  }

  err := camera.Close()
  if err != nil {
     r := api.Result{
      Successful: false,
      ErrorKind: "CouldNotCloseStream",
      Message: err.Error(),
    }
   return &r, err
  }
  delete(cameras, cameraID)

  r := api.Result{
    Successful: true,
  }
  return &r, nil
}

func main() {
  flag.Parse()
  lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *iface, *port))
  if err != nil {
    log.Fatalf("failed to listen: %v", err)
  }
  log.Printf("Listening on %s:%d", *iface, *port)

  s := CameraServer{}
  grpcServer := grpc.NewServer()
  api.RegisterCameraServer(grpcServer, &s)
  grpcServer.Serve(lis)
}
