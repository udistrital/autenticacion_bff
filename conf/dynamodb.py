import boto3
from conf.conf import get_settings

settings = get_settings()


def get_dynamodb_resource():
    kwargs = {
        "region_name": settings.AWS_REGION,
    }

    if settings.AWS_ACCESS_KEY_ID:
        kwargs["aws_access_key_id"] = settings.AWS_ACCESS_KEY_ID

    if settings.AWS_SECRET_ACCESS_KEY:
        kwargs["aws_secret_access_key"] = settings.AWS_SECRET_ACCESS_KEY

    if settings.DYNAMODB_ENDPOINT_URL:
        kwargs["endpoint_url"] = settings.DYNAMODB_ENDPOINT_URL

    return boto3.resource("dynamodb", **kwargs)


def get_sesiones_table():
    dynamodb = get_dynamodb_resource()
    return dynamodb.Table(settings.DYNAMODB_TABLE_SESIONES)
