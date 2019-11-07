use serde::*;
use json;
use sha2::Digest;
use diesel::{RunQueryDsl, ExpressionMethods};
use rayon::prelude::*;
use std::{
    time::{Instant, Duration},
    str::FromStr,
    sync::{Mutex, Arc}
};
use crate::nevra::Nevra;

#[derive(Debug, Deserialize)]
pub struct HostPackages {
    id: i32,
    arch: String,
    packages: Vec<String>,
}

pub struct Bencher {
    pub start_time: Option<Instant>,
    pub total_count: usize,
    pub saved_count: usize,
}

impl Bencher {
    fn save(&mut self) {
        let start = self.start_time.get_or_insert(Instant::now());
        self.saved_count += 1;
        if self.saved_count == self.total_count {
            let end = Instant::now();

            println!("Saved :{:?} items, {:?} item/s", self.saved_count, self.total_count as f64 / end.duration_since(start.clone()).as_secs_f64());
            self.start_time = None;
            self.saved_count = 0;
        }
    }
}

fn kafka_runner<F: FnMut(HostPackages)>(mut handler: F) {
    use rdkafka::*;
    use rdkafka::consumer::{BaseConsumer, Consumer};
    use rdkafka::config::RDKafkaLogLevel;

    let url = std::env::var("LISTENER_KAFKA_ADDRESS").unwrap();
    let topic = std::env::var("LISTENER_KAFKA_TOPIC").unwrap();

    println!("Connecting to kafka on : {:?}", url);
    let hosts = vec!(url.to_owned());

    let consumer: BaseConsumer = rdkafka::config::ClientConfig::new()
        .set("bootstrap.servers", &url)
        .set("broker.address.family", "v4")
        .set("group.id", "worker")

        .set("enable.partition.eof", "false")
        .set("session.timeout.ms", "6000")
        .set("enable.auto.commit", "true")
        .set("auto.offset.reset", "earliest")
        .set_log_level(RDKafkaLogLevel::Debug)
        .create()
        .expect("Consumer");

    consumer.subscribe(&[&topic]).unwrap();


    for ms in consumer.iter() {
        let ms = ms.unwrap();

        let data: HostPackages = json::from_slice(ms.payload_view::<[u8]>().unwrap().unwrap()).unwrap();
        handler(data);
    }
}

fn run(pool: crate::db::Pool) {

    let bench = Bencher {
        start_time: None,
        total_count: std::env::var("BENCHMARK_MESSAGES").unwrap().parse().unwrap(),
        saved_count: 0,
    };

    let bench = Arc::new(Mutex::new(bench));

    kafka_runner(|data| {
        let pool = pool.clone();
        let bench = bench.clone();

        // Spawn a task onto a thread pool.
        rayon::spawn_fifo(move || {
            let arch = data.arch;

            let mut packages: Vec<_> = data.packages.iter().map(|s| {
                Nevra::from_str(&s).unwrap()
            }).collect();

            // Drop Invalid packages
            packages.retain(|pkg| {
                pkg.arch == arch
            });

            let request = json::json!({
                    "package_list" : data.packages
                });

            let req = json::to_string(&request).unwrap();
            let mut sha = sha2::Sha256::new();
            sha.input(&req);
            let sha = sha.result();
            let checksum = hex::encode(&sha[..]);


            let value = crate::db::schema::Host {
                id: data.id,
                request: req,
                checksum: checksum,
            };

            {
                use crate::db::schema::hosts::dsl::*;
                use diesel::pg::upsert::*;

                diesel::insert_into(hosts)
                    .values(&value)
                    .on_conflict(id)
                    .do_update()
                    .set(
                        (
                            request.eq(excluded(request)),
                            checksum.eq(excluded(checksum))
                        )
                    )
                    .execute(&pool.get().unwrap()).unwrap();

                bench.lock().unwrap().save();
            }

        })
    })
}

pub fn spawn(pool: crate::db::Pool) -> std::thread::JoinHandle<()> {
    std::thread::spawn(|| {
        println!("RUNNING KAFKA");
        run(pool);
    })
}
