use crate::error::{RepoDocsError, Result};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;

pub struct GracefulShutdown {
    running: Arc<AtomicBool>,
    shutdown_message_shown: Arc<AtomicBool>,
}

impl GracefulShutdown {
    pub fn new() -> Result<Self> {
        let running = Arc::new(AtomicBool::new(true));
        let shutdown_message_shown = Arc::new(AtomicBool::new(false));

        let running_clone = running.clone();
        let message_shown_clone = shutdown_message_shown.clone();

        // Handle Ctrl+C gracefully
        ctrlc::set_handler(move || {
            running_clone.store(false, Ordering::SeqCst);

            if !message_shown_clone.swap(true, Ordering::SeqCst) {
                eprintln!("\n🛑 Gracefully stopping... (press Ctrl+C again to force exit)");
            } else {
                eprintln!("\n💀 Force stopping...");
                std::process::exit(1);
            }
        })
        .map_err(|e| RepoDocsError::Config {
            message: format!("Failed to set signal handler: {}", e),
        })?;

        Ok(Self {
            running,
            shutdown_message_shown,
        })
    }

    /// Create a GracefulShutdown instance for testing (no signal handler registration)
    pub fn new_for_test() -> Self {
        Self {
            running: Arc::new(AtomicBool::new(true)),
            shutdown_message_shown: Arc::new(AtomicBool::new(false)),
        }
    }

    pub fn is_running(&self) -> bool {
        self.running.load(Ordering::SeqCst)
    }

    pub fn check_shutdown(&self) -> Result<()> {
        if !self.is_running() {
            return Err(RepoDocsError::Cancelled);
        }
        Ok(())
    }

    pub fn request_shutdown(&self) {
        self.running.store(false, Ordering::SeqCst);
    }

    pub fn reset(&self) {
        self.running.store(true, Ordering::SeqCst);
        self.shutdown_message_shown.store(false, Ordering::SeqCst);
    }

    pub fn with_shutdown_check<F, R>(&self, operation: F) -> Result<R>
    where
        F: FnOnce() -> Result<R>,
    {
        self.check_shutdown()?;
        let result = operation()?;
        self.check_shutdown()?;
        Ok(result)
    }

    pub fn with_periodic_checks<F, R>(
        &self,
        mut operation: F,
        check_interval_ops: usize,
    ) -> Result<R>
    where
        F: FnMut(&Self) -> Result<Option<R>>,
    {
        let mut operation_count = 0;

        loop {
            // Check for shutdown periodically
            if operation_count % check_interval_ops == 0 {
                self.check_shutdown()?;
            }

            match operation(self)? {
                Some(result) => return Ok(result),
                None => {
                    operation_count += 1;
                    continue;
                }
            }
        }
    }
}

impl Default for GracefulShutdown {
    fn default() -> Self {
        Self::new().unwrap_or_else(|_| {
            // Fallback if signal handler setup fails
            Self {
                running: Arc::new(AtomicBool::new(true)),
                shutdown_message_shown: Arc::new(AtomicBool::new(false)),
            }
        })
    }
}

// Shutdown-aware operation wrapper
pub struct ShutdownAwareOperation<'a> {
    shutdown: &'a GracefulShutdown,
    #[allow(dead_code)]
    operation_name: String,
}

impl<'a> ShutdownAwareOperation<'a> {
    pub fn new(shutdown: &'a GracefulShutdown, operation_name: &str) -> Self {
        Self {
            shutdown,
            operation_name: operation_name.to_string(),
        }
    }

    pub fn execute<F, R>(&self, operation: F) -> Result<R>
    where
        F: FnOnce() -> Result<R>,
    {
        self.shutdown.check_shutdown()?;

        let result = operation().map_err(|e| {
            // If operation fails and we're shutting down, prioritize shutdown error
            if !self.shutdown.is_running() {
                RepoDocsError::Cancelled
            } else {
                e
            }
        })?;

        self.shutdown.check_shutdown()?;
        Ok(result)
    }

    pub fn execute_with_progress<F, R, P>(&self, mut operation: F) -> Result<R>
    where
        F: FnMut(&Self) -> Result<Option<(R, P)>>,
        P: std::fmt::Display,
    {
        loop {
            self.shutdown.check_shutdown()?;

            match operation(self)? {
                Some((result, _progress)) => {
                    return Ok(result);
                }
                None => {
                    // Continue operation
                    std::thread::sleep(std::time::Duration::from_millis(10));
                }
            }
        }
    }

    pub fn check_shutdown(&self) -> Result<()> {
        self.shutdown.check_shutdown()
    }

    pub fn is_running(&self) -> bool {
        self.shutdown.is_running()
    }
}

// Global shutdown coordinator for complex operations
pub struct ShutdownCoordinator {
    shutdown: Arc<GracefulShutdown>,
    active_operations: Arc<AtomicBool>,
}

impl ShutdownCoordinator {
    pub fn new() -> Result<Self> {
        Ok(Self {
            shutdown: Arc::new(GracefulShutdown::new()?),
            active_operations: Arc::new(AtomicBool::new(false)),
        })
    }

    pub fn start_operation(&self) -> Result<ShutdownAwareOperation<'_>> {
        self.shutdown.check_shutdown()?;
        self.active_operations.store(true, Ordering::SeqCst);

        Ok(ShutdownAwareOperation::new(
            &self.shutdown,
            "coordinated_operation",
        ))
    }

    pub fn finish_operation(&self) {
        self.active_operations.store(false, Ordering::SeqCst);
    }

    pub fn request_shutdown(&self) {
        self.shutdown.request_shutdown();
    }

    pub fn wait_for_operations_to_complete(&self, timeout: std::time::Duration) -> bool {
        let start = std::time::Instant::now();

        while self.active_operations.load(Ordering::SeqCst) {
            if start.elapsed() > timeout {
                return false; // Timeout
            }
            std::thread::sleep(std::time::Duration::from_millis(10));
        }

        true // Operations completed
    }

    pub fn shutdown(&self) -> Arc<GracefulShutdown> {
        self.shutdown.clone()
    }

    pub fn is_running(&self) -> bool {
        self.shutdown.is_running()
    }
}

impl Default for ShutdownCoordinator {
    fn default() -> Self {
        Self::new().unwrap_or_else(|_| Self {
            shutdown: Arc::new(GracefulShutdown::default()),
            active_operations: Arc::new(AtomicBool::new(false)),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    #[test]
    fn test_graceful_shutdown_creation() {
        // Note: This test might fail in some CI environments that don't support signal handling
        let shutdown = GracefulShutdown::new();
        match shutdown {
            Ok(shutdown) => {
                assert!(shutdown.is_running());
            }
            Err(_) => {
                // Signal handler setup failed, use default
                let shutdown = GracefulShutdown::default();
                assert!(shutdown.is_running());
            }
        }
    }

    #[test]
    fn test_shutdown_state_management() {
        let shutdown = GracefulShutdown::default();

        assert!(shutdown.is_running());
        assert!(shutdown.check_shutdown().is_ok());

        shutdown.request_shutdown();
        assert!(!shutdown.is_running());
        assert!(shutdown.check_shutdown().is_err());

        shutdown.reset();
        assert!(shutdown.is_running());
        assert!(shutdown.check_shutdown().is_ok());
    }

    #[test]
    fn test_with_shutdown_check() {
        let shutdown = GracefulShutdown::default();

        // Operation should succeed when running
        let result = shutdown.with_shutdown_check(|| Ok(42));
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), 42);

        // Operation should fail when shutdown is requested
        shutdown.request_shutdown();
        let result = shutdown.with_shutdown_check(|| Ok(42));
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), RepoDocsError::Cancelled));
    }

    #[test]
    fn test_shutdown_aware_operation() {
        let shutdown = GracefulShutdown::default();
        let operation = ShutdownAwareOperation::new(&shutdown, "test_op");

        // Should execute successfully when running
        let result = operation.execute(|| Ok("success"));
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), "success");

        // Should fail when shutdown is requested
        shutdown.request_shutdown();
        let result = operation.execute(|| Ok("success"));
        assert!(result.is_err());
    }

    #[test]
    fn test_shutdown_coordinator() {
        let coordinator = ShutdownCoordinator::default();

        assert!(coordinator.is_running());

        // Start an operation
        let operation = coordinator.start_operation().unwrap();
        assert!(operation.is_running());

        // Request shutdown
        coordinator.request_shutdown();
        assert!(!coordinator.is_running());
        assert!(!operation.is_running());

        // Finish operation
        coordinator.finish_operation();

        // Wait should return immediately since no operations are active
        assert!(coordinator.wait_for_operations_to_complete(Duration::from_millis(10)));
    }

    #[test]
    fn test_periodic_checks() {
        let shutdown = GracefulShutdown::default();
        let mut counter = 0;

        let result = shutdown.with_periodic_checks(
            |shutdown| {
                counter += 1;

                if counter >= 3 {
                    Ok(Some(counter))
                } else {
                    shutdown.check_shutdown()?;
                    Ok(None)
                }
            },
            1,
        );

        assert!(result.is_ok());
        assert_eq!(result.unwrap(), 3);
    }

    #[test]
    fn test_periodic_checks_with_shutdown() {
        let shutdown = GracefulShutdown::default();
        let mut counter = 0;

        let result = shutdown.with_periodic_checks(
            |_shutdown| {
                counter += 1;

                if counter == 2 {
                    // Request shutdown on second iteration
                    shutdown.request_shutdown();
                }

                if counter >= 5 {
                    Ok(Some(counter))
                } else {
                    Ok(None)
                }
            },
            1,
        );

        // Should be cancelled before reaching 5
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), RepoDocsError::Cancelled));
    }

    #[test]
    fn test_operation_coordination() {
        let coordinator = ShutdownCoordinator::default();

        // Test that operations can be started and coordinated
        {
            let _operation1 = coordinator.start_operation().unwrap();
            // Operation should be marked as active

            let timeout = Duration::from_millis(10);
            // Should timeout because operation is still active
            assert!(!coordinator.wait_for_operations_to_complete(timeout));
        } // operation1 drops here

        coordinator.finish_operation();

        // Now should complete immediately
        let timeout = Duration::from_millis(10);
        assert!(coordinator.wait_for_operations_to_complete(timeout));
    }
}
