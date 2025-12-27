# CI Pipeline Fixes

## Issues Identified from GitHub Actions Job

Based on the failed job at: https://github.com/quadgatefoundation/fluxor/actions/runs/20538247944/job/58999230431

### Problems Found:

1. **Cache Errors** (11 errors, 1 warning)
   - Multiple "Cannot open: File exists" errors
   - Tar restore failures: `"/usr/bin/tar" failed with error: The process '/usr/bin/tar' failed with exit code 2`
   - These errors were preventing proper cache restoration

2. **Test Failures**
   - "Run tests" step completed with exit code 1
   - Tests were failing but errors weren't clearly visible due to cache issues

## Fixes Applied

### 1. Cache Configuration Improvements

**Before:**
```yaml
- name: Cache Go modules
  uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ matrix.go-version }}-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      ${{ runner.os }}-go-${{ matrix.go-version }}-
```

**After:**
```yaml
- name: Cache Go modules
  uses: actions/cache@v4
  id: cache
  continue-on-error: true
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ matrix.go-version }}-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      ${{ runner.os }}-go-${{ matrix.go-version }}-

- name: Clean cache on error
  if: steps.cache.outputs.cache-hit != 'true'
  run: |
    rm -rf ~/.cache/go-build || true
    rm -rf ~/go/pkg/mod || true
```

**Changes:**
- Added `id: cache` to track cache step status
- Added `continue-on-error: true` to prevent cache failures from stopping the job
- Added cleanup step to remove corrupted cache directories
- Applied to both `test` and `test-examples` jobs

### 2. Test Step Improvements

**Before:**
```yaml
- name: Run tests
  run: go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

**After:**
```yaml
- name: Run tests
  timeout-minutes: 15
  run: |
    # Run tests with timeout and better error handling
    go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout 10m ./... || {
      echo "Tests failed. Checking for specific issues..."
      # List any test files that might have issues
      find . -name "*_test.go" -type f | head -20
      exit 1
    }
```

**Changes:**
- Added step-level timeout (15 minutes)
- Added test-level timeout (10 minutes) to prevent hanging tests
- Added error handling with diagnostic output
- Lists test files when failures occur for easier debugging

## Expected Results

After these fixes:

1. **Cache errors will be handled gracefully**
   - Cache failures won't stop the job
   - Corrupted cache will be cleaned up automatically
   - Dependencies will be downloaded fresh if cache fails

2. **Test failures will be more visible**
   - Better error messages
   - Test file listings for debugging
   - Timeout protection against hanging tests

3. **More reliable CI runs**
   - Jobs won't fail due to cache corruption
   - Better diagnostics when tests fail
   - Timeout protection prevents infinite hangs

## Next Steps

1. **Monitor the next CI run** to verify fixes work
2. **Check test output** if tests still fail to identify specific test issues
3. **Review test files** if needed based on error output
4. **Consider adding** test result artifacts for easier debugging

## Additional Recommendations

If tests continue to fail after cache fixes:

1. **Run tests locally** to reproduce issues:
   ```bash
   go test -v -race ./...
   ```

2. **Check for flaky tests** that might be timing-dependent

3. **Review race detector output** for potential race conditions

4. **Check Go version compatibility** - ensure all code works with Go 1.24

5. **Consider splitting test job** into multiple jobs for better isolation:
   - Unit tests
   - Integration tests
   - Race detector tests

## References

- [GitHub Actions Cache Documentation](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- [Go Test Documentation](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [Actions Cache v4](https://github.com/actions/cache)

