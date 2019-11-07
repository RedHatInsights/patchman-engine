use diesel::*;
use diesel::r2d2::*;


use diesel_migrations::*;
use diesel_migrations::run_pending_migrations;

embed_migrations!("./migrations");

fn db_url() -> String {
    let db = std::env::var("DB_TYPE").unwrap();
    let user = std::env::var("DB_USER").unwrap();
    let pass = std::env::var("DB_PASSWD").unwrap();
    let host = std::env::var("DB_HOST").unwrap();
    let name = std::env::var("DB_NAME").unwrap();

    format!("{}://{}:{}@{}/{}", db, user, pass, host, name)
}

pub type Pool = r2d2::Pool<ConnectionManager<PgConnection>>;
pub type Connection = r2d2::PooledConnection<r2d2::ConnectionManager<PgConnection>>;

pub fn connect() -> Pool {
    let manager = ConnectionManager::new(db_url());
    r2d2::Pool::new(manager).unwrap()
}

pub fn init() -> Pool {
    let pool = connect();
    let connection = pool.get().unwrap();

    run_pending_migrations(&connection).unwrap();
    return pool;
}

pub mod schema {
    use serde::{Deserialize, Serialize};

    table! {
        hosts (id) {
            id -> Int4,
            request -> Varchar,
            checksum -> Varchar,
        }
    }

    #[derive(Debug, Clone, Queryable,Insertable, AsChangeset, Deserialize, Serialize)]
    pub struct Host {
        pub id: i32,
        pub request: String,
        pub checksum: String,
    }
}
