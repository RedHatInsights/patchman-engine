"""
Postgresql settings for peewee mappings.
"""
import os

from peewee import PostgresqlDatabase

DB_NAME = os.getenv('DB_NAME', "spm")
DB_USER = os.getenv('DB_USER', "spm_admin")
DB_PASS = os.getenv('DB_PASSWD', "spm_admin_pwd")
DB_HOST = os.getenv('DB_HOST', "spm_db")
DB_PORT = int(os.getenv('DB_PORT', "5432"))

DB = PostgresqlDatabase(DB_NAME, user=DB_USER, password=DB_PASS, host=DB_HOST, port=DB_PORT)
