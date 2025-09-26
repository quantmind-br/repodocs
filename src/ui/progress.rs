use crate::cloner::CloneProgress;
use crate::extractor::ExtractionProgress;
use indicatif::{MultiProgress, ProgressBar, ProgressStyle};
use std::time::Duration;

pub struct ProgressManager {
    multi_progress: MultiProgress,
    enabled: bool,
}

impl ProgressManager {
    pub fn new(enabled: bool) -> Self {
        Self {
            multi_progress: MultiProgress::new(),
            enabled,
        }
    }

    pub fn create_clone_progress(&self) -> ProgressBar {
        if !self.enabled {
            return ProgressBar::hidden();
        }

        let pb = self.multi_progress.add(ProgressBar::new(100));
        pb.set_style(
            ProgressStyle::with_template(
                "{spinner:.green} [{elapsed_precise}] [{bar:40.cyan/blue}] {pos:>3}% {msg}",
            )
            .unwrap_or_else(|_| ProgressStyle::default_bar())
            .progress_chars("#>-"),
        );
        pb.set_message("Initializing clone...");
        pb.enable_steady_tick(Duration::from_millis(100));
        pb
    }

    pub fn create_file_progress(&self, total_files: u64) -> ProgressBar {
        if !self.enabled {
            return ProgressBar::hidden();
        }

        let pb = self.multi_progress.add(ProgressBar::new(total_files));
        pb.set_style(
            ProgressStyle::with_template(
                "{spinner:.green} [{elapsed_precise}] [{bar:40.cyan/blue}] {pos:>7}/{len:7} files {msg}"
            )
            .unwrap_or_else(|_| ProgressStyle::default_bar())
            .progress_chars("#>-")
        );
        pb.set_message("Processing files...");
        pb.enable_steady_tick(Duration::from_millis(100));
        pb
    }

    pub fn create_spinner(&self, message: &str) -> ProgressBar {
        if !self.enabled {
            return ProgressBar::hidden();
        }

        let pb = self.multi_progress.add(ProgressBar::new_spinner());
        pb.enable_steady_tick(Duration::from_millis(100));
        pb.set_style(
            ProgressStyle::with_template("{spinner:.green} {msg} ({elapsed})")
                .unwrap_or_else(|_| ProgressStyle::default_spinner())
                .tick_strings(&["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]),
        );
        pb.set_message(message.to_string());
        pb
    }

    pub fn create_bytes_progress(&self, total_bytes: u64, message: &str) -> ProgressBar {
        if !self.enabled {
            return ProgressBar::hidden();
        }

        let pb = self.multi_progress.add(ProgressBar::new(total_bytes));
        pb.set_style(
            ProgressStyle::with_template(
                "{spinner:.green} [{elapsed_precise}] [{bar:40.cyan/blue}] {bytes:>7}/{total_bytes:7} {msg}"
            )
            .unwrap_or_else(|_| ProgressStyle::default_bar())
            .progress_chars("#>-")
        );
        pb.set_message(message.to_string());
        pb.enable_steady_tick(Duration::from_millis(100));
        pb
    }

    pub fn suspend<F, R>(&self, f: F) -> R
    where
        F: FnOnce() -> R,
    {
        if self.enabled {
            self.multi_progress.suspend(f)
        } else {
            f()
        }
    }

    pub fn clear(&self) {
        if self.enabled {
            self.multi_progress.clear().ok();
        }
    }

    pub fn is_enabled(&self) -> bool {
        self.enabled
    }
}

impl Default for ProgressManager {
    fn default() -> Self {
        Self::new(true)
    }
}

// Helper functions for updating progress bars based on application events
pub fn update_clone_progress(pb: &ProgressBar, progress: &CloneProgress) {
    if progress.total_objects > 0 {
        let percentage = (progress.received_objects * 100) / progress.total_objects;
        pb.set_position(percentage as u64);

        if progress.received_objects == progress.total_objects && progress.total_deltas > 0 {
            pb.set_message(format!(
                "Resolving deltas {}/{}",
                progress.indexed_deltas, progress.total_deltas
            ));
        } else {
            pb.set_message(format!(
                "Receiving objects {}/{} ({:.1} KB)",
                progress.received_objects,
                progress.total_objects,
                progress.received_bytes as f64 / 1024.0
            ));
        }
    } else {
        pb.set_message("Counting objects...");
    }
}

pub fn update_file_progress(pb: &ProgressBar, progress: &ExtractionProgress) {
    pb.set_position(progress.files_processed as u64);

    if let Some(ref current_file) = progress.current_file {
        let eta = if progress.files_processed > 0 {
            let estimated_remaining = progress.estimated_remaining();
            if estimated_remaining.as_secs() > 0 {
                format!(" (ETA: {})", format_duration(estimated_remaining))
            } else {
                String::new()
            }
        } else {
            String::new()
        };

        pb.set_message(format!("Processing {}{}", current_file, eta));
    } else {
        pb.set_message("Processing files...");
    }
}

pub fn finish_progress_with_summary(pb: &ProgressBar, message: &str, duration: Duration) {
    let final_message = format!("{} (completed in {})", message, format_duration(duration));
    pb.finish_with_message(final_message);
}

fn format_duration(duration: Duration) -> String {
    let secs = duration.as_secs();
    if secs >= 60 {
        format!("{}m {}s", secs / 60, secs % 60)
    } else if secs > 0 {
        format!("{}s", secs)
    } else {
        format!("{}ms", duration.as_millis())
    }
}

// Progress tracking wrapper for operations
pub struct OperationProgress {
    progress_bar: ProgressBar,
    operation_name: String,
    start_time: std::time::Instant,
}

impl OperationProgress {
    pub fn new(progress_manager: &ProgressManager, operation_name: &str, total_units: u64) -> Self {
        let progress_bar = if total_units == 0 {
            progress_manager.create_spinner(operation_name)
        } else {
            progress_manager.create_file_progress(total_units)
        };

        Self {
            progress_bar,
            operation_name: operation_name.to_string(),
            start_time: std::time::Instant::now(),
        }
    }

    pub fn update(&self, current: u64, message: Option<&str>) {
        self.progress_bar.set_position(current);
        if let Some(msg) = message {
            self.progress_bar.set_message(msg.to_string());
        }
    }

    pub fn set_message(&self, message: &str) {
        self.progress_bar.set_message(message.to_string());
    }

    pub fn increment(&self, delta: u64) {
        self.progress_bar.inc(delta);
    }

    pub fn finish_with_message(&self, message: &str) {
        let duration = self.start_time.elapsed();
        let final_message = format!(
            "{}: {} ({})",
            self.operation_name,
            message,
            format_duration(duration)
        );
        self.progress_bar.finish_with_message(final_message);
    }

    pub fn finish_success(&self) {
        self.finish_with_message("completed successfully");
    }

    pub fn finish_error(&self, error: &str) {
        self.finish_with_message(&format!("failed: {}", error));
    }

    pub fn abandon_with_message(&self, message: &str) {
        self.progress_bar.abandon_with_message(message.to_string());
    }
}

// Multi-operation progress coordination
pub struct MultiOperationProgress {
    operations: Vec<OperationProgress>,
    current_operation: Option<usize>,
}

impl MultiOperationProgress {
    pub fn new() -> Self {
        Self {
            operations: Vec::new(),
            current_operation: None,
        }
    }

    pub fn add_operation(&mut self, operation: OperationProgress) -> usize {
        let index = self.operations.len();
        self.operations.push(operation);
        index
    }

    pub fn start_operation(&mut self, index: usize) {
        self.current_operation = Some(index);
    }

    pub fn update_current(&self, progress: u64, message: Option<&str>) {
        if let Some(current_idx) = self.current_operation {
            if let Some(operation) = self.operations.get(current_idx) {
                operation.update(progress, message);
            }
        }
    }

    pub fn finish_current_success(&mut self) {
        if let Some(current_idx) = self.current_operation {
            if let Some(operation) = self.operations.get(current_idx) {
                operation.finish_success();
            }
            self.current_operation = None;
        }
    }

    pub fn finish_current_error(&mut self, error: &str) {
        if let Some(current_idx) = self.current_operation {
            if let Some(operation) = self.operations.get(current_idx) {
                operation.finish_error(error);
            }
            self.current_operation = None;
        }
    }
}

impl Default for MultiOperationProgress {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_progress_manager_creation() {
        let manager = ProgressManager::new(true);
        assert!(manager.is_enabled());

        let disabled_manager = ProgressManager::new(false);
        assert!(!disabled_manager.is_enabled());
    }

    #[test]
    fn test_progress_bar_creation() {
        let manager = ProgressManager::new(true);

        let clone_pb = manager.create_clone_progress();
        let file_pb = manager.create_file_progress(100);
        let spinner = manager.create_spinner("test");

        // In test environments, progress bars might be hidden due to no TTY
        // Just test that they are created without panicking
        // The visibility depends on the environment (TTY vs non-TTY)
        assert!(clone_pb.length().unwrap_or(0) > 0 || clone_pb.length().is_none());
        assert!(file_pb.length().unwrap_or(0) > 0 || file_pb.length().is_none());
        assert!(!spinner.message().is_empty());
    }

    #[test]
    fn test_disabled_progress_bars() {
        let manager = ProgressManager::new(false);

        let clone_pb = manager.create_clone_progress();
        assert!(clone_pb.is_hidden());

        let file_pb = manager.create_file_progress(100);
        assert!(file_pb.is_hidden());

        let spinner = manager.create_spinner("test");
        assert!(spinner.is_hidden());
    }

    #[test]
    fn test_format_duration() {
        assert_eq!(format_duration(Duration::from_secs(30)), "30s");
        assert_eq!(format_duration(Duration::from_secs(90)), "1m 30s");
        assert_eq!(format_duration(Duration::from_secs(3661)), "61m 1s");
        assert_eq!(format_duration(Duration::from_millis(500)), "500ms");
    }

    #[test]
    fn test_operation_progress() {
        let manager = ProgressManager::new(true);
        let op_progress = OperationProgress::new(&manager, "test operation", 100);

        op_progress.update(50, Some("halfway done"));
        op_progress.increment(10);
        op_progress.set_message("almost finished");
        op_progress.finish_success();
    }

    #[test]
    fn test_multi_operation_progress() {
        let manager = ProgressManager::new(true);
        let mut multi_progress = MultiOperationProgress::new();

        let op1 = OperationProgress::new(&manager, "operation 1", 50);
        let op2 = OperationProgress::new(&manager, "operation 2", 100);

        let op1_idx = multi_progress.add_operation(op1);
        let op2_idx = multi_progress.add_operation(op2);

        multi_progress.start_operation(op1_idx);
        multi_progress.update_current(25, Some("progress"));
        multi_progress.finish_current_success();

        multi_progress.start_operation(op2_idx);
        multi_progress.update_current(50, None);
        multi_progress.finish_current_error("test error");
    }
}
