package worker

import (
	"bytes"

	gocontext "context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/travis-ci/worker/config"
)

// A BuildTracePersister persists a build trace. (duh)
type BuildTracePersister interface {
	Persist(gocontext.Context, Job, []byte) error
}

// TODO: explicit accessKeyID, secretAccessKey like build cache

type s3BuildTracePersister struct {
	bucket    string
	keyPrefix string
	region    string
}

// NewBuildTracePersister creates a build trace persister backed by S3
func NewBuildTracePersister(cfg *config.Config) BuildTracePersister {
	if !cfg.BuildTraceEnabled {
		return nil
	}

	return &s3BuildTracePersister{
		bucket:    cfg.BuildTraceS3Bucket,
		keyPrefix: cfg.BuildTraceS3KeyPrefix,
		region:    cfg.BuildTraceS3Region,
	}
}

// TODO: cache aws session for reuse? goroutine pool?
// TODO: handle job restarts -- archive separate trace per run? use job UUID?
//       perhaps we can rely on s3 versioning for this.

func (p *s3BuildTracePersister) Persist(ctx gocontext.Context, job Job, buf []byte) error {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(p.region)})
	if err != nil {
		return err
	}

	key := p.keyPrefix + "/" + string(job.Payload().Job.ID)

	_, err = s3.New(sess).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(p.bucket),
		Key:                  aws.String(key),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buf),
		ContentLength:        aws.Int64(int64(len(buf))),
		ContentType:          aws.String("application/octet-stream"),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})

	return err
}
