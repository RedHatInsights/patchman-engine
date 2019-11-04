"""
Postgresql settings for peewee mappings.
"""
import os

from peewee import PostgresqlDatabase

DB_NAME = os.getenv('POSTGRESQL_DATABASE', "vulnerability")
DB_USER = os.getenv('POSTGRESQL_USER', "ve_db_user_unknown")
DB_PASS = os.getenv('POSTGRESQL_PASSWORD', "ve_db_user_unknown_pwd")
DB_HOST = os.getenv('POSTGRESQL_HOST', "ve_database")
DB_PORT = int(os.getenv('POSTGRESQL_PORT', "5432"))

DB = PostgresqlDatabase(DB_NAME, user=DB_USER, password=DB_PASS, host=DB_HOST, port=DB_PORT)
