#! /usr/bin/env bash
set -euo pipefail
# set -x

#####################
#                   #
# DECLARE CONSTANTS #
#                   #
#####################

KLUTCHIO_REPO="$(git rev-parse --show-toplevel)"
CURRENT_BRANCH_KLUTCHIO="$(cd "$KLUTCHIO_REPO" && git rev-parse --abbrev-ref HEAD)"
readonly KLUTCHIO_REPO
readonly BIND_SUBFOLDER="$KLUTCHIO_REPO/bind"
readonly CROSSPLANE_API_SUBFOLDER="$KLUTCHIO_REPO/crossplane-api"
readonly PROVIDER_ANYNINES_SUBFOLDER="$KLUTCHIO_REPO/provider-anynines"
readonly E2E_TESTS_SUBFOLDER="$KLUTCHIO_REPO/test/e2e"
readonly DOCS_REPO="$KLUTCHIO_REPO/../klutchio-website"
readonly ECR_REGISTRY_ADDRESS="public.ecr.aws/w5n9a2g2"
readonly CURRENT_BRANCH_KLUTCHIO

#####################
#                   #
# DECLARE FUNCTIONS #
#                   #
#####################

log_info() {
    echo "[$(date "+%H:%M:%S")]" "$@"
}

log_fatal_error() {
    echo -e "[$(date "+%H:%M:%S")]"'\033[31m ERROR\033[0m:' "$@"
    exit 1
}

check_dependencies() {
    for cmd in "$@"; do
        log_info "Checking for dependency: $cmd"
        if ! command -v "$cmd" 1>/dev/null 2>&1; then
            log_fatal_error "$cmd not found in \$PATH"
        fi
    done
}

print_usage() {
    log_info "Usage: create_release.sh <version number> [-p <AWS profile name>] [-b <git branch name>]"
}

check_flag_single_use() {
    local flagName="$1"
    local flagVariable="$2"
    if [[ ! -z ${!flagVariable:-} ]]; then
        log_fatal_error "flag $flagName cannot be used multiple times\n$(print_usage)"
    fi
}

parse_arguments() {
    unset SUPPLIED_AWS_PROFILE
    unset GIT_BRANCH

    while [[ $# -gt 0 ]]; do
        case "$1" in
        -p | --profile)
            check_flag_single_use "$1" "SUPPLIED_AWS_PROFILE"
            SUPPLIED_AWS_PROFILE="$2"
            shift 2
            ;;
        -b | --branch)
            check_flag_single_use "$1" "GIT_BRANCH"
            GIT_BRANCH="$2"
            shift 2
            ;;
        *)
            # if VERSION_NUMBER is not set, we set it here. Otherwise, we treat unknown arguments as an error.
            if [[ -z ${VERSION_NUMBER:-} ]]; then
                VERSION_NUMBER="$1"
                shift
                continue
            fi
            if [[ $1 =~ ^- ]]; then
                log_fatal_error "unknown option $1\n$(print_usage)"
            fi
            log_fatal_error "too many arguments\n$(print_usage)"
            ;;
        esac
    done

    if [[ -z ${VERSION_NUMBER:-} ]]; then
        log_fatal_error "version number is required\n$(print_usage)"
    fi
}

init() {
    parse_arguments "$@"

    VERSION_CHECK_EXPRESSION='^v[0-9]\.[0-9]\.[0-9](-[-.A-z0-9]+)?$'
    if ! echo "$VERSION_NUMBER" | grep -E "$VERSION_CHECK_EXPRESSION" -q; then
        log_fatal_error "illegal version number $VERSION_NUMBER.\n\nPlease use a version number that matches this regular expression: $VERSION_CHECK_EXPRESSION"
    fi

    if [[ ! -d $DOCS_REPO ]]; then
        log_fatal_error "../klutchio-website does not exist.\nPlease make sure that the klutchio-website repo is cloned into the parent directory of the klutchio repo with that name."
    fi

    CURRENT_BRANCH_DOCS="$(cd "$DOCS_REPO" && git rev-parse --abbrev-ref HEAD)"

    local message
    if [[ -z ${SUPPLIED_AWS_PROFILE:-} ]]; then
        if [[ -z ${AWS_PROFILE:-} ]]; then
            log_info "Detecting AWS account name..."
            AWS_PROFILE="$(aws configure list | grep "profile" | cut -d ':' -f 2 | awk '{$1=$1};1')"
            export AWS_PROFILE
        fi

        message="No AWS profile specified, using active AWS profile $AWS_PROFILE"
    else
        export AWS_PROFILE="$SUPPLIED_AWS_PROFILE"
        message="Using supplied AWS profile $AWS_PROFILE"
    fi

    log_info "$message for interacting with the ECR registry $ECR_REGISTRY_ADDRESS"

    check_dependencies "ko" "crossplane" "git" "aws" "yq" "make" "gh"
}

ensure_remote_git_branch() {
    if ! git remote | grep -q "origin"; then
        log_fatal_error "No git remotes origin found. Please add a git remote with that name and retry."
    fi

    if git branch -a | grep -q -E '^[ *] remotes/origin/'"$GIT_BRANCH"'$'; then
        return
    fi

    log_info "No remote branch found for $GIT_BRANCH, creating it"
    git push --set-upstream origin "$GIT_BRANCH"
}

setup_git_branch_in_directory() {
    cd "$1"
    log_info "Setting up git branch $GIT_BRANCH in directory ${1}..."

    CHECKOUT_FLAG=""
    if ! (git branch -a | grep -q -E '^[ *] (remotes/.+/)?'"$GIT_BRANCH"'$'); then
        CHECKOUT_FLAG="-b"
    fi
    git stash --all
    git checkout $CHECKOUT_FLAG "$GIT_BRANCH"

    ensure_remote_git_branch
}

setup_git_branches() {
    trap 'cleanup' EXIT
    if [[ -z ${GIT_BRANCH:-} ]]; then
        GIT_BRANCH="$CURRENT_BRANCH_KLUTCHIO"
        log_info "No git branch specified, using current branch $CURRENT_BRANCH_KLUTCHIO"
        ensure_remote_git_branch
        return
    fi

    setup_git_branch_in_directory "$KLUTCHIO_REPO"
    setup_git_branch_in_directory "$DOCS_REPO"
}

build_docker_images() {
    log_info "Logging into AWS ECR..."
    aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin "${ECR_REGISTRY_ADDRESS}"

    log_info "Building konnector image..."
    KO_DOCKER_REPO="${ECR_REGISTRY_ADDRESS}/klutch/konnector" ko build "$BIND_SUBFOLDER/cmd/konnector" --bare -t "${VERSION_NUMBER}" --platform=linux/amd64,linux/arm64

    log_info "Building example-backend image..."
    KO_DOCKER_REPO="${ECR_REGISTRY_ADDRESS}/klutch/example-backend" ko build "$BIND_SUBFOLDER/cmd/example-backend" --bare -t "${VERSION_NUMBER}" --platform=linux/amd64,linux/arm64

    log_info "Building and pushing dataservices configuration image..."
    make -C "$CROSSPLANE_API_SUBFOLDER" dataservices-config-push dataservicesConfigVersion="${VERSION_NUMBER}"

    cd "$PROVIDER_ANYNINES_SUBFOLDER"

    log_info "Building provider-anynines controller images..."
    make buildx.all IMAGETAG="${VERSION_NUMBER}"

    log_info "Building provider-anynines Provider image..."
    make provider-build-push IMAGETAG="${VERSION_NUMBER}"
}

build_klutch-bind_binaries() {
    log_info "Building klutch-bind binaries for all platforms..."
    make -C "$BIND_SUBFOLDER"
}

update_manifests() {
    log_info "Updating version references in klutchio manifests and README.md..."
    yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/dataservices:$VERSION_NUMBER\"" \
        "$CROSSPLANE_API_SUBFOLDER/deploy/config-pkg-anynines.yaml"
    yq -i "with(select(document_index == 0); .spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\")" \
        "$CROSSPLANE_API_SUBFOLDER/deploy/provider-anynines.yaml"

    README_PATH="$CROSSPLANE_API_SUBFOLDER/README.md"
    sed -E 's|klutch/dataservices:v[0-9]+\.[0-9]+\.[0-9]+[^"]*|klutch/dataservices:'"${VERSION_NUMBER}"'|g' \
        "$README_PATH" >"${README_PATH}.tmp"
    mv "${README_PATH}.tmp" "$README_PATH"

    yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\"" \
        "$PROVIDER_ANYNINES_SUBFOLDER/examples/provider/provider.yaml"
    yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\"" \
        "$E2E_TESTS_SUBFOLDER/provider/manifests/install/provider.yaml"
    yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/dataservices:$VERSION_NUMBER\"" \
        "$E2E_TESTS_SUBFOLDER/provider/manifests/configuration.yaml"
    log_info "Klutchio manifest and README.md version references updated to ${VERSION_NUMBER}.\n"
}

try_with_tolerated_error_string() {
    toleratedErrorString="$1"
    shift
    set +e
    output=("$("$@")")
    returnCode=$?
    set -e
    if [[ $returnCode -ne 0 ]]; then
        if ! echo "${output[*]}" | grep -q -F "$toleratedErrorString"; then
            log_info "Command $* failed with error:\n${output[*]}"
            exit 1
        fi
    fi
    echo "${output[@]}"
}

commit_and_push_changes() {
    repoName="$1"
    shift

    log_info "Committing and pushing changes in $repoName with message: $*"

    cd "$repoName"
    git add --all
    try_with_tolerated_error_string "nothing to commit" git commit -m "$@"
    try_with_tolerated_error_string "Everything up-to-date" git push origin "$GIT_BRANCH"
}

update_changelog() {
    # The ":1" in the variable expansion instructs the shell to omit the first character (the leading "v") from the version number when inserting it into the changelog.
    HEADLINE="[${VERSION_NUMBER:1}] - $(date "+%Y-%m-%d")"
    if grep -x -q -F "## $HEADLINE" "$KLUTCHIO_REPO/CHANGELOG.md"; then
        log_info "Changelog already contains a section for version $VERSION_NUMBER. Skipping..."
        return
    fi

    log_info "Replacing 'Unreleased' changelog section header with header '$HEADLINE'..."
    sed -E 's/^## Unreleased$/'"## $HEADLINE/g" \
        "$KLUTCHIO_REPO/CHANGELOG.md" >"$KLUTCHIO_REPO/CHANGELOG.md.tmp"
    mv "$KLUTCHIO_REPO/CHANGELOG.md.tmp" "$KLUTCHIO_REPO/CHANGELOG.md"
}

update_klutchio_repo() {
    update_manifests
    update_changelog
    commit_and_push_changes "$KLUTCHIO_REPO" "Prepare version $VERSION_NUMBER"
    recreate_changelog_unreleased_section
}

recreate_changelog_unreleased_section() {
    if grep -q "^## Unreleased$" "$KLUTCHIO_REPO/CHANGELOG.md"; then
        log_info 'Changelog already contains an "Unreleased" section. Skipping recreation of "Unreleased" section...'
        return
    fi

    log_info 'Recreating "Unreleased" section in Changelog...'
    echo -e "# CHANGELOG\n\n## Unreleased\n$(tail -n +2 "$KLUTCHIO_REPO/CHANGELOG.md")" >"$KLUTCHIO_REPO/CHANGELOG.md"
    commit_and_push_changes "$KLUTCHIO_REPO" "Recreate Changelog Section for unreleased Changes"
}

update_documentation_repo() {
    log_info "Updating version references in klutchio-website documentation repo"
    sed -E 's|klutch/konnector:v[0-9]+\.[0-9]+\.[0-9]+[^ ]*|klutch/konnector:'"${VERSION_NUMBER}"'|g' \
        "$DOCS_REPO/docs/local-deployment-guide.md" >"$DOCS_REPO/docs/local-deployment-guide.md.tmp"
    mv "$DOCS_REPO/docs/local-deployment-guide.md.tmp" "$DOCS_REPO/docs/local-deployment-guide.md"

    sed -E 's|klutch/provider-anynines:v[0-9]+\.[0-9]+\.[0-9]+[^ ]*|klutch/provider-anynines:'"${VERSION_NUMBER}"'|g' \
        "$DOCS_REPO/docs/platform-operator-guide/monitor-klutch-components.md" > \
        "$DOCS_REPO/docs/platform-operator-guide/monitor-klutch-components.md.tmp"
    mv "$DOCS_REPO/docs/platform-operator-guide/monitor-klutch-components.md.tmp" \
        "$DOCS_REPO/docs/platform-operator-guide/monitor-klutch-components.md"

    sed -E 's|https://anynines-artifacts.s3.eu-central-1.amazonaws.com/central-management/v[0-9]+\.[0-9]+\.[0-9]+[^/]*|https://anynines-artifacts.s3.eu-central-1.amazonaws.com/central-management/'"${VERSION_NUMBER}"'|g' \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md" > \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md.tmp"
    mv "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md.tmp" \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md"

    sed -E 's|klutch/konnector:v[0-9]+\.[0-9]+\.[0-9]+[^ ]*|klutch/konnector:'"${VERSION_NUMBER}"'|g' \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md" > \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md.tmp"
    mv "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md.tmp" \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md"

    sed -E 's|klutch/example-backend:v[0-9]+\.[0-9]+\.[0-9]+[^ ]*|klutch/example-backend:'"${VERSION_NUMBER}"'|g' \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md" > \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md.tmp"
    mv "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md.tmp" \
        "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md"

    commit_and_push_changes "$DOCS_REPO" "Update documentation to reference version $VERSION_NUMBER of klutchio components"
}

cleanup() {
    log_info "Cleaning up git branches and stashes..."
    if [[ -d $DOCS_REPO ]]; then
        cd "$DOCS_REPO"
        git checkout "$CURRENT_BRANCH_DOCS"
    fi
    cd "$KLUTCHIO_REPO"
    git checkout "$CURRENT_BRANCH_KLUTCHIO"
    git stash apply -q 2>/dev/null
}

################################
#                              #
# SCRIPT EXECUTION STARTS HERE #
#                              #
################################

init "$@"
build_docker_images
build_klutch-bind_binaries
setup_git_branches
update_klutchio_repo
update_documentation_repo
