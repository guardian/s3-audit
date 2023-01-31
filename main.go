package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	/*
		Q. What is a 'public' bucket?

		AWS provide a definition here:

		https://docs.aws.amazon.com/AmazonS3/latest/userguide/access-control-block-public-access.html#access-control-block-public-access-policy-status

		There are two parts:

		1) does the ACL grant access to 'AllUsers' or 'AuthenticatedUsers'?
		2) does the bucket policy grant any access to a 'non-fixed' value

		For (2) a 'non-fixed' value includes things like use of '*' in the
		policy. But see the link above for a fuller definition.

		Q. How to list public buckets

		A number of approaches suggest themselves:

		1) Just try and read a bucket object

		Add a (tiny) object and then attempt to read it (without credentials).
		If bucket policy is set to public, the object will also be public.

		This is a narrower definition of 'public' than AWS recognise. It also
		assumes there is an object that can be read so will not work for empty
		buckets or buckets where the only objects have been individually
		restricted.

		2) AWS Access Analyzer

		https://docs.aws.amazon.com/AmazonS3/latest/userguide/access-analyzer.html

		AWS provide this analysis themselves. There is even a CLI.

		Let's do both! Note, we want to do this for all accounts and all regions
		:(. It will run on my local machine so Janus credentials are sufficient
		for now.
	*/

	ctx := context.TODO()

	config, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("eu-west-1"),
		config.WithSharedConfigProfile("deployTools"),
	)
	check(err, "unable to load AWS config")

	client := s3.NewFromConfig(config)
	buckets, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	check(err, "unable to list buckets")

	for _, bucket := range buckets.Buckets {
		if canGetObject(client, *bucket.Name) {
			fmt.Printf("%-60s\tpublic", *bucket.Name)
		} else {
			fmt.Printf("%-60s\tprivate", *bucket.Name)
		}

		println()
	}
}

func canGetObject(client *s3.Client, bucketName string) bool {
	key, err := putObject(client, bucketName, strings.NewReader("test-please-delete-this-file"))
	if err != nil {
		//log.Printf("unable to write to %s: %v", bucketName, err)
		return false
	}
	defer deleteObject(client, bucketName, key)

	return headObject(client, bucketName, key) == nil
}

func putObject(client *s3.Client, bucketName string, data io.Reader) (string, error) {
	randKey := "sldkfjsldkfjslkdjfsdlkfjiwe"

	_, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &randKey,
		Body:   data,
	})

	return randKey, err
}

func headObject(client *s3.Client, bucketName string, key string) error {
	url := fmt.Sprintf("https://%s.s3.eu-west-1.amazonaws.com/%s", bucketName, key)
	resp, err := http.Head(url)

	if resp.StatusCode != http.StatusOK {
		//log.Printf("unable to get s3://%s/%s: %v", bucketName, key, resp.StatusCode)
		return errors.New(strconv.Itoa(resp.StatusCode))
	}

	return err

	/*
		 	return client.HeadObject(context.TODO(), &s3.HeadObjectInput{
				Bucket: &bucketName,
				Key:    &key,
			})
	*/
}

func deleteObject(client *s3.Client, bucketName string, key string) (*s3.DeleteObjectOutput, error) {
	return client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{Bucket: &bucketName, Key: &key})
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}
