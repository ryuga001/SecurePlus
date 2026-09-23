package provider

import (
	"context"
	"errors"

	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
)

const ProviderAWS = "aws"

func (c *Client) AWSCallerIdentity(ctx context.Context, accessKeyID, secretAccessKey, region string) error {
	client := sts.New(sts.Options{
		Region:      region,
		Credentials: awscreds.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		HTTPClient:  c.http,
	})

	if _, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err != nil {
		return awsError(err)
	}

	return nil
}

func awsError(err error) error {
	var api smithy.APIError
	if errors.As(err, &api) {
		return &Error{
			Provider: ProviderAWS,
			Stage:    StageToken,
			Reason:   awsReason(api.ErrorCode()),
			code:     api.ErrorCode(),
		}
	}

	return &Error{
		Provider: ProviderAWS,
		Stage:    StageToken,
		Reason:   ReasonUnavailable,
	}
}

func awsReason(code string) string {
	switch code {
	case "InvalidClientTokenId", "SignatureDoesNotMatch", "IncompleteSignature", "MissingAuthenticationToken":
		return ReasonAuthFailed
	case "AccessDenied", "AccessDeniedException":
		return ReasonPermissionDenied
	case "Throttling", "ThrottlingException", "TooManyRequestsException":
		return ReasonRateLimited
	}

	return ReasonUnknown
}
