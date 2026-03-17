from psycopg import connect
from psycopg.rows import dict_row

from conf.conf import get_settings

settings = get_settings()


def get_connection():
    return connect(
        host=settings.DB_HOST,
        port=settings.DB_PORT,
        dbname=settings.DB_NAME,
        user=settings.DB_USER,
        password=settings.DB_PASSWORD,
        row_factory=dict_row,
    )
