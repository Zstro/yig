package _go

import (
	. "github.com/journeymidnight/yig/test/go/lib"
	"testing"
)

const (
	TEST_METADATA_UNDERLINE_KEY = "x-amz-meta-test_underline"
	TEST_METADATA_NORMAL_KEY    = "x-amz-meta-test-Normal"
)

func Test_addMetadataNormal(t *testing.T) {
	sc := NewS3()
	sc.CleanEnv()
	err := sc.MakeBucket(TEST_BUCKET)
	if err != nil {
		t.Fatal("MakeBucket err:", err)
		panic(err)
	}
	metadata := make(map[string]string)

	metadata[TEST_METADATA_NORMAL_KEY] = "normal key value"

	err = sc.PutObjectWithMetadata(TEST_BUCKET, TEST_KEY, TEST_VALUE, metadata)
	if err != nil {
		t.Fatal("PutObject err:", err)
	}
	sc.CleanEnv()
}

func Test_addMetadataUnderline(t *testing.T) {
	sc := NewS3()
	sc.CleanEnv()
	err := sc.MakeBucket(TEST_BUCKET)
	if err != nil {
		t.Fatal("MakeBucket err:", err)
		panic(err)
	}
	metadata := make(map[string]string)

	metadata[TEST_METADATA_UNDERLINE_KEY] = "underline key value"

	err = sc.PutObjectWithMetadata(TEST_BUCKET, TEST_KEY, TEST_VALUE, metadata)
	if err != nil {
		t.Fatal("PutObject err:", err)
	}
	sc.CleanEnv()
}