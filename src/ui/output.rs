use crate::error::{RepoDocsError, UserFriendlyError};
use crate::extractor::{ExtractionProgress, ExtractionReport};
use console::{style, Emoji, Term};
use serde_json;
use std::time::Duration;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum OutputMode {
    Human,
    Json,
    Plain,
}

impl OutputMode {
    pub fn from_string(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "json" => OutputMode::Json,
            "plain" => OutputMode::Plain,
            _ => OutputMode::Human,
        }
    }
}

// Emojis with text fallbacks
static CHECKMARK: Emoji = Emoji("✅ ", "✓ ");
static CROSS: Emoji = Emoji("❌ ", "✗ ");
static INFO: Emoji = Emoji("ℹ️  ", "i ");
static WARNING: Emoji = Emoji("⚠️  ", "! ");
static ROCKET: Emoji = Emoji("🚀 ", "> ");
static SPARKLES: Emoji = Emoji("✨ ", "* ");

pub struct OutputFormatter {
    #[allow(dead_code)]
    term: Term,
    mode: OutputMode,
    use_colors: bool,
    verbose_level: u8,
    quiet: bool,
}

impl OutputFormatter {
    pub fn new(mode: OutputMode, verbose: u8, quiet: bool) -> Self {
        let term = Term::stdout();
        let use_colors = match mode {
            OutputMode::Human => term.features().colors_supported() && !quiet,
            _ => false,
        };

        Self {
            term,
            mode,
            use_colors,
            verbose_level: if quiet { 0 } else { verbose },
            quiet,
        }
    }

    // Core messaging methods
    pub fn success(&self, message: &str) {
        match self.mode {
            OutputMode::Human => self.print_human_message(MessageType::Success, message),
            OutputMode::Json => self.print_json_message("success", message),
            OutputMode::Plain => println!("SUCCESS: {}", message),
        }
    }

    pub fn error(&self, message: &str) {
        match self.mode {
            OutputMode::Human => self.print_human_message(MessageType::Error, message),
            OutputMode::Json => self.print_json_message("error", message),
            OutputMode::Plain => eprintln!("ERROR: {}", message),
        }
    }

    pub fn warning(&self, message: &str) {
        if self.should_show_message(1) {
            match self.mode {
                OutputMode::Human => self.print_human_message(MessageType::Warning, message),
                OutputMode::Json => self.print_json_message("warning", message),
                OutputMode::Plain => println!("WARNING: {}", message),
            }
        }
    }

    pub fn info(&self, message: &str) {
        if self.should_show_message(1) {
            match self.mode {
                OutputMode::Human => self.print_human_message(MessageType::Info, message),
                OutputMode::Json => self.print_json_message("info", message),
                OutputMode::Plain => println!("INFO: {}", message),
            }
        }
    }

    pub fn debug(&self, message: &str) {
        if self.should_show_message(2) {
            match self.mode {
                OutputMode::Human => {
                    if self.use_colors {
                        println!("  {}", style(message).dim());
                    } else {
                        println!("  DEBUG: {}", message);
                    }
                }
                OutputMode::Json => self.print_json_message("debug", message),
                OutputMode::Plain => println!("DEBUG: {}", message),
            }
        }
    }

    pub fn start_operation(&self, operation: &str) {
        if self.should_show_message(0) {
            match self.mode {
                OutputMode::Human => {
                    if self.use_colors {
                        println!("{}{}", ROCKET, style(operation).bold());
                    } else {
                        println!("> {}", operation);
                    }
                }
                OutputMode::Json => self.print_json_message("operation_start", operation),
                OutputMode::Plain => println!("STARTING: {}", operation),
            }
        }
    }

    // User-friendly error handling
    pub fn print_user_friendly_error(&self, error: &RepoDocsError) {
        let user_message = error.user_message();
        self.error(&user_message);

        if let Some(suggestion) = error.suggestion() {
            match self.mode {
                OutputMode::Human => {
                    println!();
                    if self.use_colors {
                        println!(
                            "{}{}",
                            INFO,
                            style(&format!("Suggestion: {}", suggestion)).cyan()
                        );
                    } else {
                        println!("Suggestion: {}", suggestion);
                    }
                }
                OutputMode::Json => {
                    self.print_json_object(&serde_json::json!({
                        "type": "suggestion",
                        "message": suggestion
                    }));
                }
                OutputMode::Plain => {
                    println!("SUGGESTION: {}", suggestion);
                }
            }
        }
    }

    // Summary and reporting
    pub fn print_extraction_summary(&self, progress: &ExtractionProgress) {
        if self.quiet {
            return;
        }

        match self.mode {
            OutputMode::Human => self.print_human_summary(progress),
            OutputMode::Json => self.print_json_summary(progress),
            OutputMode::Plain => self.print_plain_summary(progress),
        }
    }

    pub fn print_extraction_report(&self, report: &ExtractionReport) {
        match self.mode {
            OutputMode::Human => self.print_human_report(report),
            OutputMode::Json => {
                let json_output =
                    serde_json::to_string_pretty(report).unwrap_or_else(|_| "{}".to_string());
                println!("{}", json_output);
            }
            OutputMode::Plain => self.print_plain_report(report),
        }
    }

    // Specialized output methods
    pub fn print_header(&self, title: &str) {
        if self.quiet {
            return;
        }

        match self.mode {
            OutputMode::Human => {
                println!();
                if self.use_colors {
                    println!("{} {}", SPARKLES, style(title).bold().cyan());
                } else {
                    println!("=== {} ===", title);
                }
                println!();
            }
            OutputMode::Json => {
                self.print_json_object(&serde_json::json!({
                    "type": "header",
                    "title": title
                }));
            }
            OutputMode::Plain => {
                println!("=== {} ===", title);
            }
        }
    }

    pub fn print_separator(&self) {
        if self.quiet {
            return;
        }

        match self.mode {
            OutputMode::Human => {
                if self.use_colors {
                    println!("{}", style("─".repeat(60)).dim());
                } else {
                    println!("{}", "-".repeat(60));
                }
            }
            OutputMode::Plain => {
                println!("{}", "-".repeat(60));
            }
            OutputMode::Json => {} // No separator in JSON mode
        }
    }

    // Private helper methods
    fn should_show_message(&self, min_verbose_level: u8) -> bool {
        !self.quiet && self.verbose_level >= min_verbose_level
    }

    fn print_human_message(&self, msg_type: MessageType, message: &str) {
        #[allow(clippy::type_complexity)]
        let (emoji, color_fn): (Emoji, Box<dyn Fn(&str) -> console::StyledObject<&str>>) =
            match msg_type {
                MessageType::Success => (CHECKMARK, Box::new(|msg| style(msg).green().bold())),
                MessageType::Error => (CROSS, Box::new(|msg| style(msg).red().bold())),
                MessageType::Warning => (WARNING, Box::new(|msg| style(msg).yellow().bold())),
                MessageType::Info => (INFO, Box::new(|msg| style(msg).cyan())),
            };

        if self.use_colors {
            match msg_type {
                MessageType::Error => eprintln!("{}{}", emoji, color_fn(message)),
                _ => println!("{}{}", emoji, color_fn(message)),
            }
        } else {
            let prefix = match msg_type {
                MessageType::Success => "✓",
                MessageType::Error => "✗",
                MessageType::Warning => "!",
                MessageType::Info => "i",
            };

            match msg_type {
                MessageType::Error => eprintln!("{} {}", prefix, message),
                _ => println!("{} {}", prefix, message),
            }
        }
    }

    fn print_json_message(&self, level: &str, message: &str) {
        self.print_json_object(&serde_json::json!({
            "type": "message",
            "level": level,
            "message": message,
            "timestamp": chrono::Utc::now().to_rfc3339()
        }));
    }

    fn print_json_object(&self, obj: &serde_json::Value) {
        println!(
            "{}",
            serde_json::to_string(obj).unwrap_or_else(|_| "{}".to_string())
        );
    }

    fn print_human_summary(&self, progress: &ExtractionProgress) {
        println!();
        self.print_separator();

        if self.use_colors {
            println!(
                "{} {}",
                style("Documentation extraction completed!").green().bold(),
                CHECKMARK
            );
        } else {
            println!("✓ Documentation extraction completed!");
        }

        println!();
        println!(
            "  Files processed: {}",
            if self.use_colors {
                style(progress.files_processed).cyan().bold().to_string()
            } else {
                progress.files_processed.to_string()
            }
        );
        println!(
            "  Bytes processed: {}",
            if self.use_colors {
                style(format_bytes(progress.bytes_processed))
                    .cyan()
                    .bold()
                    .to_string()
            } else {
                format_bytes(progress.bytes_processed)
            }
        );
        println!(
            "  Time taken:      {}",
            if self.use_colors {
                style(format_duration(progress.elapsed()))
                    .cyan()
                    .bold()
                    .to_string()
            } else {
                format_duration(progress.elapsed())
            }
        );

        if !progress.errors.is_empty() {
            println!("  Errors:          {}", progress.errors.len());
        }

        self.print_separator();
    }

    fn print_json_summary(&self, progress: &ExtractionProgress) {
        let summary = serde_json::json!({
            "type": "summary",
            "files_processed": progress.files_processed,
            "bytes_processed": progress.bytes_processed,
            "duration_ms": progress.elapsed().as_millis(),
            "errors": progress.errors.len(),
            "timestamp": chrono::Utc::now().to_rfc3339()
        });

        println!(
            "{}",
            serde_json::to_string_pretty(&summary).unwrap_or_else(|_| "{}".to_string())
        );
    }

    fn print_plain_summary(&self, progress: &ExtractionProgress) {
        println!("COMPLETED: Documentation extraction");
        println!("Files processed: {}", progress.files_processed);
        println!("Bytes processed: {}", progress.bytes_processed);
        println!("Duration: {:?}", progress.elapsed());
        if !progress.errors.is_empty() {
            println!("Errors: {}", progress.errors.len());
        }
    }

    fn print_human_report(&self, report: &ExtractionReport) {
        self.print_header("Extraction Report");

        println!(
            "Repository: {}/{}",
            report.repository_info.owner, report.repository_info.name
        );
        println!("URL: {}", report.repository_info.url);
        println!(
            "Extracted at: {}",
            report.extraction_time.format("%Y-%m-%d %H:%M UTC")
        );
        println!();

        if !report.extraction_summary.files_by_extension.is_empty() {
            println!("Files by type:");
            let mut extensions: Vec<_> = report
                .extraction_summary
                .files_by_extension
                .iter()
                .collect();
            extensions.sort_by(|a, b| b.1.cmp(a.1));

            for (ext, count) in extensions {
                let display_ext = if ext == "no_extension" {
                    "no extension"
                } else {
                    ext
                };
                println!("  {}: {} files", display_ext, count);
            }
            println!();
        }

        if !report.errors.is_empty() {
            println!("Issues encountered:");
            for error in &report.errors {
                println!("  - {}", error);
            }
        }
    }

    fn print_plain_report(&self, report: &ExtractionReport) {
        println!("REPORT: Extraction completed");
        println!(
            "Repository: {}/{}",
            report.repository_info.owner, report.repository_info.name
        );
        println!("Files: {}", report.extraction_summary.total_files_processed);
        println!(
            "Size: {} bytes",
            report.extraction_summary.total_bytes_processed
        );
        println!(
            "Duration: {:?}",
            report.extraction_summary.extraction_duration
        );

        if !report.errors.is_empty() {
            println!("Errors: {}", report.errors.len());
        }
    }
}

#[derive(Debug, Clone, Copy)]
enum MessageType {
    Success,
    Error,
    Warning,
    Info,
}

fn format_bytes(bytes: u64) -> String {
    const UNITS: &[&str] = &["B", "KB", "MB", "GB"];
    let mut size = bytes as f64;
    let mut unit_index = 0;

    while size >= 1024.0 && unit_index < UNITS.len() - 1 {
        size /= 1024.0;
        unit_index += 1;
    }

    if unit_index == 0 {
        format!("{} {}", bytes, UNITS[unit_index])
    } else {
        format!("{:.1} {}", size, UNITS[unit_index])
    }
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

// Progress-aware output wrapper
pub struct ProgressAwareOutput<'a> {
    formatter: &'a OutputFormatter,
    progress_manager: Option<&'a crate::ui::ProgressManager>,
}

impl<'a> ProgressAwareOutput<'a> {
    pub fn new(
        formatter: &'a OutputFormatter,
        progress_manager: Option<&'a crate::ui::ProgressManager>,
    ) -> Self {
        Self {
            formatter,
            progress_manager,
        }
    }

    pub fn suspend_and_print<F>(&self, f: F)
    where
        F: FnOnce(&OutputFormatter),
    {
        if let Some(pm) = self.progress_manager {
            pm.suspend(|| f(self.formatter));
        } else {
            f(self.formatter);
        }
    }

    pub fn success(&self, message: &str) {
        self.suspend_and_print(|f| f.success(message));
    }

    pub fn error(&self, message: &str) {
        self.suspend_and_print(|f| f.error(message));
    }

    pub fn warning(&self, message: &str) {
        self.suspend_and_print(|f| f.warning(message));
    }

    pub fn info(&self, message: &str) {
        self.suspend_and_print(|f| f.info(message));
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_output_mode_parsing() {
        assert_eq!(OutputMode::from_string("human"), OutputMode::Human);
        assert_eq!(OutputMode::from_string("json"), OutputMode::Json);
        assert_eq!(OutputMode::from_string("plain"), OutputMode::Plain);
        assert_eq!(OutputMode::from_string("invalid"), OutputMode::Human);
    }

    #[test]
    fn test_formatter_creation() {
        let formatter = OutputFormatter::new(OutputMode::Human, 1, false);
        assert_eq!(formatter.mode, OutputMode::Human);
        assert_eq!(formatter.verbose_level, 1);
        assert!(!formatter.quiet);
    }

    #[test]
    fn test_quiet_mode() {
        let formatter = OutputFormatter::new(OutputMode::Human, 2, true);
        assert_eq!(formatter.verbose_level, 0);
        assert!(formatter.quiet);
    }

    #[test]
    fn test_format_bytes() {
        assert_eq!(format_bytes(512), "512 B");
        assert_eq!(format_bytes(1024), "1.0 KB");
        assert_eq!(format_bytes(1536), "1.5 KB");
        assert_eq!(format_bytes(1048576), "1.0 MB");
        assert_eq!(format_bytes(1073741824), "1.0 GB");
    }

    #[test]
    fn test_format_duration() {
        assert_eq!(format_duration(Duration::from_secs(30)), "30s");
        assert_eq!(format_duration(Duration::from_secs(90)), "1m 30s");
        assert_eq!(format_duration(Duration::from_secs(3661)), "61m 1s");
        assert_eq!(format_duration(Duration::from_millis(500)), "500ms");
        assert_eq!(format_duration(Duration::from_millis(0)), "0ms");
    }

    #[test]
    fn test_should_show_message() {
        let formatter = OutputFormatter::new(OutputMode::Human, 2, false);
        assert!(formatter.should_show_message(0));
        assert!(formatter.should_show_message(1));
        assert!(formatter.should_show_message(2));
        assert!(!formatter.should_show_message(3));

        let quiet_formatter = OutputFormatter::new(OutputMode::Human, 2, true);
        assert!(!quiet_formatter.should_show_message(0));
        assert!(!quiet_formatter.should_show_message(1));
        assert!(!quiet_formatter.should_show_message(2));
    }
}
