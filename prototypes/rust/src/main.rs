#[macro_use]
extern crate diesel;

#[macro_use]
extern crate diesel_migrations;

#[macro_use]
extern crate lazy_static;


use actix_web::{web, http, App, Responder, Either};
use std::time::Duration;
use diesel::*;
use actix_web::web::Json;

pub mod nevra;
pub mod db;
pub mod listener;

use crate::db::schema::{ReadHost};
use sha2::Digest;

fn single_host((host_id, db): (web::Path<i32>, web::Data<db::Pool>)) -> impl Responder {
    let conn = db.get().unwrap();
    use db::schema::hosts;
    let host_id = host_id.into_inner();

    let host: Result<ReadHost, _> = hosts::table.filter(hosts::id.eq(&host_id)).first(&conn);
    let host = match host {
        Err(diesel::NotFound) => {
            return Either::A(web::HttpResponse::new(http::StatusCode::NOT_FOUND));
        }
        r => {
            r.unwrap()
        }
    };

    let mut sha = sha2::Sha256::new();
    sha.input(&host.request);
    let sha = sha.result();
    let checksum = hex::encode(&sha[..]);

    if checksum != host.checksum {
        return Either::A(web::HttpResponse::new(http::StatusCode::PARTIAL_CONTENT));
    }

    Either::B(Json(host))
}

fn all_hosts(db: web::Data<db::Pool>) -> impl Responder {
    let conn = db.get().unwrap();
    use db::schema::hosts;

    let hosts: Result<Vec<ReadHost>, diesel::result::Error> =  hosts::table.load(&conn);
    //dbg!(&hosts);

    let hosts = match hosts {
        Err(diesel::NotFound) => {
            return Either::A(web::HttpResponse::new(http::StatusCode::NOT_FOUND));
        }
        r => {
            r.unwrap()
        }
    };

    Either::B(Json(hosts))
}

fn delete_all_hosts(pool: &db::Pool) {
    use crate::db::schema::hosts::dsl::*;

    diesel::delete(hosts)
        .execute(&pool.get().unwrap()).unwrap();
}

fn main() {
    std::env::set_var("RUST_LOG", "prototype=trace,info");
    std::env::set_var("RUST_BACKTRACE", "full");
    env_logger::init();
    std::thread::sleep(Duration::from_secs(4));

    let pool = db::init();

    delete_all_hosts(&pool);

    let list = listener::spawn(pool.clone());

    println!("RUNNING WEB");

    actix_web::HttpServer::new(move || {
        App::new()
            .data(pool.clone())
            .service(
                web::resource("/hosts/{id}").to(single_host)
            ).service(web::resource("/hosts").to(all_hosts))
    })
        .bind("0.0.0.0:8082")
        .unwrap()
        .run()
        .unwrap();

    list.join().unwrap();
}
