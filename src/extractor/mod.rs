pub mod file_extractor;
pub mod output_manager;

pub use file_extractor::{FileOperations, ExtractionProgress};
pub use output_manager::{OutputManager, ExtractionReport, ConfigSnapshot};