## BQ Write API using protobuf

This repo demonstrates generating a BQ table using a `.proto` as the schema and then inserting data into that table using the [BQ Storage Write API](https://cloud.google.com/bigquery/docs/write-api).

Data from the client will be in protobuf format and we will use the [AppendRowsRequest.ProtoData](https://cloud.google.com/bigquery/docs/reference/storage/rpc/google.cloud.bigquery.storage.v1#google.cloud.bigquery.storage.v1.AppendRowsRequest.ProtoData) function marshall and insert the source data.   

This repo heavily lifts code from the two repos shown below but the variation described here is that we will define the _schema_ directly through a `protoc` compiler.

References:

- [alexflint/bigquery-storage-api-example](https://github.com/alexflint/bigquery-storage-api-example)
- [SO: Using BigQuery Write API in Golang](https://stackoverflow.com/questions/70279279/using-bigquery-write-api-in-golang)

---

The first step is to convert our sample `.proto` to BQ shema format.

given the following proto, 

```proto
syntax = "proto3";

package echo;
option go_package = "github.com/salrashid123/grpc_wireformat/grpc_services/src/echo";

message EchoRequest {
  string first_name = 1;
  string last_name = 2;
  Middle middle_name = 3;
}

message Middle {
  string name = 1;
}

message EchoReply {
  string message = 1;
}
```

We will first use the protoc plugin [protoc-gen-bq-schema](https://github.com/GoogleCloudPlatform/protoc-gen-bq-schema) to help convert it.

so first install the plutgin (you'll ofcourse need `golang` and `protoc` setup on your system)

```bash
go install github.com/GoogleCloudPlatform/protoc-gen-bq-schema@latest
git clone https://github.com/GoogleCloudPlatform/protoc-gen-bq-schema.git
```


This specific compiler uses annotations overrides so we will add in the imports and annotations

```proto
syntax = "proto3";

package echo;
option go_package = "github.com/salrashid123/grpc_wireformat/grpc_services/src/echo";

import "bq_table.proto";
import "bq_field.proto";

message EchoRequest {
  option (gen_bq_schema.bigquery_opts).table_name = "echorequest";
  string first_name = 1;
  string last_name = 2;
  Middle middle_name = 3;
}

message Middle {
  string name = 1;
}

message EchoReply {
  string message = 1;
}
```

Now generate the schema:

```bash
cp protoc-gen-bq-schema/bq_field.proto .
cp protoc-gen-bq-schema/bq_table.proto .
protoc --go_out=. --bq-schema_out=. --go_opt=paths=source_relative   --descriptor_set_out=echo/echo.pb     echo/echo.proto
```

You should see the format BQ expects


```bash
$ cat echo/echorequest.schema 
[
 {
  "name": "first_name",
  "type": "STRING",
  "mode": "NULLABLE"
 },
 {
  "name": "last_name",
  "type": "STRING",
  "mode": "NULLABLE"
 },
 {
  "name": "middle_name",
  "type": "RECORD",
  "mode": "NULLABLE",
  "fields": [
   {
    "name": "name",
    "type": "STRING",
    "mode": "NULLABLE"
   }
  ]
 }
]
```


Now create a BQ table with that schema

```bash
export GCLOUD_USER=`gcloud config get-value core/account`
export PROJECT_ID=`gcloud config get-value core/project`
export DATASET_ID=echo_dataset
export TABLE_ID=echorequest
export PROJECT_NUMBER=`gcloud projects describe $PROJECT_ID --format='value(projectNumber)'`

bq mk -d --data_location=US $DATASET_ID
bq mk   -t   --description "prototable  Keys" $DATASET_ID.$TABLE_ID  echo/echorequest.schema
```

Use the writeAPI to insert a row of data where the source data is actually a full go protobuf Message

```bash
go run bq_client.go --projectID $PROJECT_ID --datasetID $DATASET_ID --tableID $TABLE_ID
```


If you query the table now

```bash
bq  query \
--use_legacy_sql=false  "SELECT * 
FROM $PROJECT_ID.$DATASET_ID.$TABLE_ID
LIMIT 5;"

+------------+-----------+--------------+
| first_name | last_name | middle_name  |
+------------+-----------+--------------+
| sal        | mander    | {"name":"a"} |
+------------+-----------+--------------+
```


---

and just for amusement, 

* [BigQuery UDF Marshall/Unmarshall Protocolbuffers](https://github.com/salrashid123/bq-udf-protobuf)
* [gRPC Unary requests the hard way: using protorefelect, dynamicpb and wire-encoding to send messages](https://blog.salrashid.dev/articles/2022/grpc_wireformat/)