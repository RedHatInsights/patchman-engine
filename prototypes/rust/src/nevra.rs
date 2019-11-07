use serde::*;
use regex::*;

use std::str::FromStr;


#[derive(Debug, Clone, PartialEq, PartialOrd, Eq, Ord, Hash)]
pub struct Nevra {
    pub name: String,
    pub epoch: Option<String>,
    pub version: String,
    pub release: String,
    pub arch: String,
}


impl ToString for Nevra {
    fn to_string(&self) -> String {
        let epoch = if let Some(ref epoch) = self.epoch {
            format!("{}:", epoch)
        } else {
            String::new()
        };

        format!(
            "{}-{}{}-{}.{}",
            self.name, epoch, self.version, self.release, self.arch
        )
    }
}

impl Serialize for Nevra {
    fn serialize<S>(&self, serializer: S) -> Result<<S as Serializer>::Ok, <S as Serializer>::Error>
        where
            S: Serializer,
    {
        serializer.serialize_str(&self.to_string())
    }
}

impl<'de> Deserialize<'de> for Nevra {
    fn deserialize<D>(deserializer: D) -> Result<Self, <D as Deserializer<'de>>::Error> where
        D: Deserializer<'de> {

        let txt = String::deserialize(deserializer)?;

        Ok(Self::from_str(&txt).map_err(|_| serde::de::Error::custom("Invalid nevra"))?)

    }
}

pub const PKG_NAME : &str ="([^:(/=<> ]+)";
pub const PKG_EPOCH : &str = "([0-9]+:)?";
pub const PKG_VERSION : &str = "([^-:(/=<> ]+)" ;
pub const PKG_RELEASE : &str = PKG_VERSION;
pub const PKG_ARCH : &str = "([^-:.(/=<> ]+)";

lazy_static! {
    static ref NEVRA_RE: Regex =
    //Regex::new(&format!(r#"^{}-{}{}-{}\.{}$"#, PKG_NAME,PKG_EPOCH, PKG_VERSION, PKG_RELEASE, PKG_ARCH)).unwrap();
        Regex::new(r#"^(.*)-([0-9]+:)?([^-]+)-([^-]+)\.([a-z0-9_]+)$"#).unwrap();
}

impl FromStr for Nevra {
    type Err = ();

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        // TODO: Rewrite using nom parser, gonna be faster and prettier
        if let Some(caps) = NEVRA_RE.captures(s) {
            return Ok(Nevra {
                name: caps.get(1).map(|x| x.as_str().to_owned()).unwrap(),
                epoch: caps.get(2).map(|x| x.as_str().to_owned()),
                version: caps.get(3).map(|x| x.as_str().to_owned()).unwrap(),
                release: caps.get(4).map(|x| x.as_str().to_owned()).unwrap(),
                arch: caps.get(5).map(|x| x.as_str().to_owned()).unwrap(),
            });
        }
        Err(())
    }
}

#[test]
fn test_nevra() {
    let nevra = Nevra::from_str("389-ds-base-1.3.7.8-1.fc27.src").unwrap();
    assert_eq!("389-ds-base", nevra.name);
    assert_eq!("1.3.7.8", nevra.version);
    assert_eq!("1.fc27", nevra.release);
    assert_eq!("src", nevra.arch);
}