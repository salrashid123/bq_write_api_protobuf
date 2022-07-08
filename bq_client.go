package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	storage "cloud.google.com/go/bigquery/storage/apiv1beta2"
	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"github.com/salrashid123/grpc_wireformat/grpc_services/src/echo"
	storagepb "google.golang.org/genproto/googleapis/cloud/bigquery/storage/v1beta2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const ()

var (
	projectID = flag.String("projectID", "foo", "projectID")
	datasetID = flag.String("datasetID", "echo_dataset", "datasetID")
	tableID   = flag.String("tableID", "echorequest", "tableID")
)

func main() {

	flag.Parse()

	row := &echo.EchoRequest{
		FirstName: "sal",
		MiddleName: &echo.Middle{
			Name: "a",
		},
		LastName: "mander",
	}

	b, err := protojson.Marshal(row)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", string(b))

	ctx := context.Background()

	// create the bigquery client
	client, err := storage.NewBigQueryWriteClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	resp, err := client.CreateWriteStream(ctx, &storagepb.CreateWriteStreamRequest{
		Parent: fmt.Sprintf("projects/%s/datasets/%s/tables/%s", *projectID, *datasetID, *tableID),
		WriteStream: &storagepb.WriteStream{
			Type: storagepb.WriteStream_COMMITTED,
		},
	})
	if err != nil {
		log.Fatal("CreateWriteStream: ", err)
	}

	rows := []echo.EchoRequest{}
	rows = append(rows, *row)

	// much of the following is from
	// - [alexflint/bigquery-storage-api-example](https://github.com/alexflint/bigquery-storage-api-example)
	// - [SO: Using BigQuery Write API in Golang](https://stackoverflow.com/questions/70279279/using-bigquery-write-api-in-golang)

	stream, err := client.AppendRows(ctx)
	if err != nil {
		log.Fatal("AppendRows: ", err)
	}

	descriptor, err := adapt.NormalizeDescriptor(row.ProtoReflect().Descriptor())
	if err != nil {
		log.Fatal("NormalizeDescriptor: ", err)
	}

	var opts proto.MarshalOptions
	var data [][]byte
	for _, r := range rows {
		buf, err := opts.Marshal(r.ProtoReflect().Interface())
		if err != nil {
			log.Fatal("protobuf.Marshal: ", err)
		}
		data = append(data, buf)
	}

	err = stream.Send(&storagepb.AppendRowsRequest{
		WriteStream: resp.Name,
		Rows: &storagepb.AppendRowsRequest_ProtoRows{
			ProtoRows: &storagepb.AppendRowsRequest_ProtoData{
				WriterSchema: &storagepb.ProtoSchema{
					ProtoDescriptor: descriptor,
				},
				Rows: &storagepb.ProtoRows{
					SerializedRows: data,
				},
			},
		},
	})
	if err != nil {
		log.Fatal("AppendRows.Send: ", err)
	}

	_, err = stream.Recv()
	if err != nil {
		log.Fatal("AppendRows.Recv: ", err)
	}

	log.Println("done")

}
