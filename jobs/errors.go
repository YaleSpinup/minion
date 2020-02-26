package jobs

import (
	"github.com/YaleSpinup/minion/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case
			// Access denied.
			"AccessDenied",

			// There is a problem with your AWS account that prevents the operation from completing successfully.
			"AccountProblem",

			// All access to this Amazon S3 resource has been disabled.
			"AllAccessDisabled",

			// Access forbidden.
			"Forbidden",

			// The AWS access key ID you provided does not exist in our records.
			"InvalidAccessKeyId":

			return apierror.New(apierror.ErrForbidden, msg, aerr)
		case
			// ErrCodeBucketAlreadyExists for service response error code
			// "BucketAlreadyExists".
			//
			// The requested bucket name is not available. The bucket namespace is shared
			// by all users of the system. Please select a different name and try again.
			s3.ErrCodeBucketAlreadyExists,

			// ErrCodeBucketAlreadyOwnedByYou for service response error code
			// "BucketAlreadyOwnedByYou".
			s3.ErrCodeBucketAlreadyOwnedByYou,

			// 	The bucket you tried to delete is not empty.
			"BucketNotEmpty",

			// The request is not valid with the current state of the bucket.
			"InvalidBucketState",

			// A conflicting conditional operation is currently in progress against this resource. Try again.
			"OperationAborted",

			// Object restore is already in progress.
			"RestoreAlreadyInProgress":
			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			// ErrCodeNoSuchBucket for service response error code
			// "NoSuchBucket".
			//
			// The specified bucket does not exist.
			s3.ErrCodeNoSuchBucket,

			// ErrCodeNoSuchKey for service response error code
			// "NoSuchKey".
			//
			// The specified key does not exist.
			s3.ErrCodeNoSuchKey,

			// ErrCodeNoSuchUpload for service response error code
			// "NoSuchUpload".
			//
			// The specified multipart upload does not exist.
			s3.ErrCodeNoSuchUpload,

			// The specified bucket does not exist.
			"NotFound",

			// The specified bucket does not have a bucket policy.
			"NoSuchBucketPolicy",

			// The lifecycle configuration does not exist.
			"NoSuchLifecycleConfiguration",

			// Indicates that the version ID specified in the request does not match an existing version.
			"NoSuchVersion":
			return apierror.New(apierror.ErrNotFound, msg, aerr)

		case
			// ErrCodeObjectAlreadyInActiveTierError for service response error code
			// "ObjectAlreadyInActiveTierError".
			//
			// This operation is not allowed against this storage tier
			s3.ErrCodeObjectAlreadyInActiveTierError,

			// ErrCodeObjectNotInActiveTierError for service response error code
			// "ObjectNotInActiveTierError".
			//
			// The source object of the COPY operation is not in the active tier and is
			// only stored in Amazon Glacier.
			s3.ErrCodeObjectNotInActiveTierError,

			// The email address you provided is associated with more than one account.
			"AmbiguousGrantByEmailAddress",

			// The authorization header you provided is invalid.
			"AuthorizationHeaderMalformed",

			// The Content-MD5 you specified did not match what we received.
			"BadDigest",

			// This request does not support credentials.
			"CredentialsNotSupported",

			// Cross-location logging not allowed. Buckets in one geographic location cannot log information to a bucket in another location.
			"CrossLocationLoggingProhibited",

			// Your proposed upload is smaller than the minimum allowed object size.
			"EntityTooSmall",

			// Your proposed upload exceeds the maximum allowed object size.
			"EntityTooLarge",

			// The provided token has expired.
			"ExpiredToken",

			// Indicates that the versioning configuration specified in the request is invalid.
			"IllegalVersioningConfigurationException",

			// You did not provide the number of bytes specified by the Content-Length HTTP header
			"IncompleteBody",

			// POST requires exactly one file upload per request.
			"IncorrectNumberOfFilesInPostRequest",

			// Inline data exceeds the maximum allowed size.
			"InlineDataTooLarge",

			// You must specify the Anonymous role.
			"InvalidAddressingHeader",

			// Invalid Argument
			"InvalidArgument",

			// The specified bucket is not valid.
			"InvalidBucketName",

			// 	The Content-MD5 you specified is not valid.
			"InvalidDigest",

			// The encryption request you specified is not valid. The valid value is AES256.
			"InvalidEncryptionAlgorithmError",

			// The operation is not valid for the current state of the object.
			"InvalidObjectState",

			// The specified location constraint is not valid. For more information about Regions.
			"InvalidLocationConstraint",

			// One or more of the specified parts could not be found. The part might not have been uploaded, or the
			// specified entity tag might not have matched the part's entity tag.
			"InvalidPart",

			// The list of parts was not in ascending order. Parts list must be specified in order by part number.
			"InvalidPartOrder",

			// The content of the form does not meet the conditions specified in the policy document.
			"InvalidPolicyDocument",

			// The requested range cannot be satisfied.
			"InvalidRange",

			// Please use AWS4-HMAC-SHA256.
			// SOAP requests must be made over an HTTPS connection.
			// Amazon S3 Transfer Acceleration is not supported for buckets with non-DNS compliant names.
			// Amazon S3 Transfer Acceleration is not supported for buckets with periods (.) in their names.
			// Amazon S3 Transfer Accelerate endpoint only supports virtual style requests.
			// Amazon S3 Transfer Accelerate is not configured on this bucket.
			// Amazon S3 Transfer Accelerate is disabled on this bucket.
			// Amazon S3 Transfer Acceleration is not supported on this bucket. Contact AWS Support for more information.
			// Amazon S3 Transfer Acceleration cannot be enabled on this bucket. Contact AWS Support for more information.
			"InvalidRequest",

			// The SOAP request body is invalid.
			"InvalidSOAPRequest",

			// The storage class you specified is not valid.
			"InvalidStorageClass",

			// The target bucket for logging does not exist, is not owned by you, or does not have the appropriate grants for the log-delivery group.
			"InvalidTargetBucketForLogging",

			//The provided token is malformed or otherwise invalid.
			"InvalidToken",

			// Couldn't parse the specified URI.
			"InvalidURI",

			// Your key is too long.
			"KeyTooLongError",

			// The XML you provided was not well-formed or did not validate against our published schema.
			"MalformedACLError",

			// The body of your POST request is not well-formed multipart/form-data.
			"MalformedPOSTRequest",

			// This happens when the user sends malformed XML (XML that doesn't conform to the published XSD) for the configuration.
			// The error message is, "The XML you provided was not well-formed or did not validate against our published schema."
			"MalformedXML",

			// The specified method is not allowed against this resource.
			"MethodNotAllowed",

			// A SOAP attachment was expected, but none were found.
			"MissingAttachment",

			// You must provide the Content-Length HTTP header.
			"MissingContentLength",

			// This happens when the user sends an empty XML document as a request. The error message is, "Request body is empty."
			"MissingRequestBodyError",

			// The SOAP 1.1 request is missing a security element.
			"MissingSecurityElement",

			// Your request is missing a required header.
			"MissingSecurityHeader",

			// There is no such thing as a logging status subresource for a key.
			"NoLoggingStatusForKey",

			// At least one of the preconditions you specified did not hold.
			"PreconditionFailed",

			// Bucket POST must be of the enclosure-type multipart/form-data.
			"RequestIsNotMultiPartContent",

			// Requesting the torrent file of a bucket is not permitted.
			"RequestTorrentOfBucketError",

			// The request signature we calculated does not match the signature you provided.
			// Check your AWS secret access key and signing method.
			"SignatureDoesNotMatch",

			// The provided token must be refreshed.
			"TokenRefreshRequired",

			// This request does not support content.
			"UnexpectedContent",

			// The email address you provided does not match any account on record.
			"UnresolvableGrantByEmailAddress",

			// The bucket POST must contain the specified field name. If it is specified, check the order of the fields.
			"UserKeyMustBeSpecified":

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			// Your request was too big.
			"MaxMessageLengthExceeded",

			// Your POST request fields preceding the upload file were too large.
			"MaxPostPreDataLengthExceededError",

			// Your metadata headers exceed the maximum allowed metadata size.
			"MetadataTooLarge",

			// Reduce your request rate.
			"ServiceUnavailable",

			// Reduce your request rate.
			"SlowDown",

			// You have attempted to create more buckets than allowed.
			"TooManyBuckets":

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		case
			// All access to this object has been disabled. Please contact AWS Support for further assistance.
			"InvalidPayer",

			// We encountered an internal error. Please try again.
			"InternalError",

			// The provided security credentials are not valid.
			"InvalidSecurity",

			// A header you provided implies functionality that is not implemented.
			"NotImplemented",

			// Your account is not signed up for the Amazon S3 service. You must sign up before you can use Amazon S3.
			// You can sign up at the following URL: https://aws.amazon.com/s3
			"NotSignedUp",

			// The bucket you are attempting to access must be addressed using the specified endpoint. Send all future requests to this endpoint.
			"PermanentRedirect",

			// Temporary redirect.
			"Redirect",

			// Your socket connection to the server was not read from or written to within the timeout period.
			"RequestTimeout",

			// The difference between the request time and the server's time is too large.
			"RequestTimeTooSkewed",

			// You are being redirected to the bucket while DNS updates.
			"TemporaryRedirect":
			return apierror.New(apierror.ErrServiceUnavailable, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	log.Warnf("uncaught error: %s, returning Internal Server Error", err)
	return apierror.New(apierror.ErrInternalError, msg, err)
}
