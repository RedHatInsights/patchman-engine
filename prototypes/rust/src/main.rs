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

use crate::db::schema::Host;
use sha2::Digest;

fn single_host((host_id, db): (web::Path<i32>, web::Data<db::Pool>)) -> impl Responder {
    let conn = db.get().unwrap();
    use db::schema::hosts;
    let host_id = host_id.into_inner();

    let host: Host = hosts::table.filter(hosts::id.eq(&host_id)).first(&conn).unwrap();

    let mut sha = sha2::Sha256::new();
    sha.input(&host.request);
    let sha = sha.result();
    let checksum = hex::encode(&sha[..]);

    if checksum != host.checksum {
        return Either::A(web::HttpResponse::new(http::StatusCode::PARTIAL_CONTENT));
    }

    Either::B(Ok::<_, actix_web::Error>(Json(host)))
}

fn main() {
    std::env::set_var("RUST_LOG", "prototype=trace,info");
    std::env::set_var("RUST_BACKTRACE", "full");
    env_logger::init();
    std::thread::sleep(Duration::from_secs(4));

    let pool = db::init();

    let list = listener::spawn(pool.clone());

    println!("RUNNING WEB");


    actix_web::HttpServer::new(move || {
        App::new()
            .data(pool.clone())
            .service(
                web::resource("/host/{id}").to(single_host)
            )
    })
        .bind("0.0.0.0:8082")
        .unwrap()
        .run()
        .unwrap();

    list.join().unwrap();
}
