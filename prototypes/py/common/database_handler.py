"""
Module containing database handler class.
"""
import os
from traceback import extract_tb
from psycopg2.pool import ThreadedConnectionPool
from psycopg2 import InterfaceError, DatabaseError
from common.logging import init_logging, get_logger

DB_NAME = os.getenv('POSTGRESQL_DATABASE', "vulnerability")
DB_USER = os.getenv('POSTGRESQL_USER', "ve_db_user_unknown")
DB_PASS = os.getenv('POSTGRESQL_PASSWORD', "ve_db_user_unknown_pwd")
DB_HOST = os.getenv('POSTGRESQL_HOST', "ve_database")
DB_PORT = int(os.getenv('POSTGRESQL_PORT', "5432"))

LOGGER = get_logger(__name__)
init_logging()


class NamedCursor:
    """Wrapper class for named cursor."""

    def __init__(self, db_connection, name="default"):
        self.cursor = db_connection.cursor(name=name)

    def __enter__(self):
        return self.cursor

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.cursor.close()


class DatabasePoolHandler:
    """Static class maintaining PostgreSQL connection pool."""

    pool = None
    pool_size = 0

    @classmethod
    def create_connection_pool(cls, size):
        """Create database connections."""
        cls.pool_size = size
        cls.pool = ThreadedConnectionPool(size, size,
                                          database=DB_NAME, user=DB_USER, password=DB_PASS,
                                          host=DB_HOST, port=DB_PORT)

    @classmethod
    def get_connection(cls):
        """Get database connection. Create new connection if doesn't exist."""
        return cls.pool.getconn()

    @classmethod
    def return_connection(cls, conn, close=False):
        """Return database connection to pool."""
        cls.pool.putconn(conn, close=close)

    @classmethod
    def close_all_connections(cls):
        """Close all connections."""
        cls.pool.closeall()


class DatabasePool:
    """Context manager for connection pool."""

    def __init__(self, size):
        DatabasePoolHandler.create_connection_pool(size)

    def __enter__(self):
        pass

    def __exit__(self, *_):
        DatabasePoolHandler.close_all_connections()


class DatabasePoolConnection:
    """Context manager for pooled database connection."""

    def __init__(self):
        self.conn = DatabasePoolHandler.get_connection()

    def __enter__(self):
        return self.conn

    def __exit__(self, exception_type, exception_value, traceback):
        DatabasePoolHandler.return_connection(self.conn)
        if exception_type is InterfaceError or exception_type is DatabaseError:
            LOGGER.error("Exception type: %s. Exception value: %s. Traceback: %s", exception_type, exception_value,
                         str(extract_tb(traceback)))
            LOGGER.info("Trying to recover connection...")
            DatabasePoolHandler.create_connection_pool(DatabasePoolHandler.pool_size)
            LOGGER.info("Connection recovered")
            return True
        return False
