#! /usr/bin/env bash
set -euo pipefail
set -x

#####################
#                   #
# DECLARE CONSTANTS #
#                   #
#####################

KLUTCHIO_REPO="$(git rev-parse --show-toplevel)"
readonly KLUTCHIO_REPO
readonly DOCS_REPO="$KLUTCHIO_REPO/../klutchio-website"
readonly ECR_REGISTRY_ADDRESS="public.ecr.aws/w5n9a2g2"
CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
readonly CURRENT_BRANCH

#####################
#                   #
# DECLARE FUNCTIONS #
#                   #
#####################

log_normal() {
    echo "[$(date "+%H:%M:%S")]" "$@"
}

log_error() {
    echo -e "[$(date "+%H:%M:%S")]"'\033[31m ERROR\033[0m:' "$@"
}

check_dependency() {
    for cmd in "$@"; do
        log_normal "Checking for dependency: $cmd"
        if ! command -v "$cmd" 1>/dev/null 2>&1; then
            log_error "$cmd not found in \$PATH"
            exit 1
        fi
    done
}

print_usage() {
    log_normal "Usage: create_release.sh <version number> [-p <AWS profile name>] [-b <git branch name>]"
}

check_flag_single_use() {
    local flagName="$1"
    local flagVariable="$2"
    if [[ ! -z ${!flagVariable:-} ]]; then
        log_error "flag $flagName cannot be used multiple times"
        print_usage
        exit 1
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
                log_error "unknown option $1"
                print_usage
                exit 1
            fi
            log_error "too many arguments"
            print_usage
            exit 1
            ;;
        esac
    done

    if [[ -z ${VERSION_NUMBER:-} ]]; then
        log_error "version number is required"
        print_usage
        exit 1
    fi
}

init() {
    parse_arguments "$@"

    VERSION_CHECK_EXPRESSION='^v[0-9]\.[0-9]\.[0-9](-[-.A-z0-9]+)?$'
    if ! echo "$VERSION_NUMBER" | grep -E "$VERSION_CHECK_EXPRESSION" -q; then
        log_error "illegal version number $VERSION_NUMBER.\n\nPlease use a version number that matches this regular expression: $VERSION_CHECK_EXPRESSION"
        exit 1
    fi

    if [[ ! -d $DOCS_REPO ]]; then
        log_error "../klutchio-website does not exist.\n\nPlease make sure that the klutchio-website repo is cloned into the parent directory of the klutchio repo with that name."
        exit 1
    fi

    local message
    if [[ -z ${SUPPLIED_AWS_PROFILE:-} ]]; then
        if [[ -z ${AWS_PROFILE:-} ]]; then
            log_normal "Detecting AWS account name..."
            AWS_PROFILE="$(aws configure list | grep "profile" | cut -d ':' -f 2 | awk '{$1=$1};1')"
            export AWS_PROFILE
        fi

        message="No AWS profile specified, using active AWS profile $AWS_PROFILE"
    else
        export AWS_PROFILE="$SUPPLIED_AWS_PROFILE"
        message="Using supplied AWS profile $AWS_PROFILE"
    fi

    log_normal "$message for interacting with the ECR registry $ECR_REGISTRY_ADDRESS"

    check_dependency "ko" "crossplane" "git" "aws" "yq" "make"
}

ensure_remote_git_branch() {
    if [[ -z $(git remote) ]]; then
        log_error "No git remotes found to track for branch $GIT_BRANCH. Please add a git remote and retry."
    fi

    if git branch -a | grep -q -E '^[ *] remotes/origin/'"$GIT_BRANCH"'$'; then
        GIT_REMOTE_BRANCH="remotes/origin/$GIT_BRANCH"
        GIT_REMOTE_NAME="origin"
        return
    fi

    if git branch -a | grep -q -E '^[ *] remotes/.+/'"$GIT_BRANCH"'$'; then
        GIT_REMOTE_BRANCH="$(git branch -a | grep -E '^[ *] remotes/.+/'"$GIT_BRANCH" | head -n 1)"
        GIT_REMOTE_NAME="$(echo "$GIT_REMOTE_BRANCH" | sed -E 's|^remotes/(.+)/.*$|\1|')"
        return
    fi

    log_normal "No remote branch found for $GIT_BRANCH, creating it"

    GIT_REMOTE_NAME="origin"
    if ! git remote | grep -q "^origin$"; then
        GIT_REMOTE_NAME="$(git remote | head -n 1)"
    fi

    git push --set-upstream "$GIT_REMOTE_NAME" "$GIT_BRANCH"
    # GIT_REMOTE_BRANCH="remotes/$GIT_REMOTE_NAME/$GIT_BRANCH"
}

setup_git_branch() {
    if [[ -z ${GIT_BRANCH:-} ]]; then
        GIT_BRANCH="$CURRENT_BRANCH"
        log_normal "No git branch specified, using current branch $CURRENT_BRANCH"
        ensure_remote_git_branch
        return
    fi

    log_normal "Using supplied git branch $GIT_BRANCH"

    CHECKOUT_FLAG=""
    if ! git branch -a | grep -q -E '^[ *] (remotes/.+/)?'"$GIT_BRANCH"'$'; then
        CHECKOUT_FLAG="-b"
    fi
    git checkout $CHECKOUT_FLAG "$GIT_BRANCH"

    ensure_remote_git_branch
}

build_docker_images() {
    log_normal "Logging into AWS ECR..."
    aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin "${ECR_REGISTRY_ADDRESS}"
    log_normal "ECR login successful"

    cd "$KLUTCHIO_REPO/bind"

    log_normal "Building konnector image..."
    KO_DOCKER_REPO="${ECR_REGISTRY_ADDRESS}/klutch/konnector" ko build ./cmd/konnector --bare -t "${VERSION_NUMBER}" --platform=linux/amd64,linux/arm64
    log_normal "konnector image built successfully"

    log_normal "Building example-backend image..."
    KO_DOCKER_REPO="${ECR_REGISTRY_ADDRESS}/klutch/example-backend" ko build ./cmd/example-backend --bare -t "${VERSION_NUMBER}" --platform=linux/amd64,linux/arm64
    log_normal "example-backend image built successfully"

    make -C "$KLUTCHIO_REPO/crossplane-api" dataservices-config-push dataservicesConfigVersion="${VERSION_NUMBER}"

    cd "$KLUTCHIO_REPO/provider-anynines"
    log_normal "Building provider-anynines controller images..."
    make buildx.all IMAGETAG="${VERSION_NUMBER}"
    log_normal "provider-anynines controller images built successfully"
    log_normal "Building provider-anynines Provider image..."
    make provider-build-push IMAGETAG="${VERSION_NUMBER}"
    log_normal "provider-anynines Provider image built successfully"
}

update_manifests() {
    log_normal "Updating version references in klutchio manifests and README.md..."
    yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/dataservices:$VERSION_NUMBER\"" \
        "$KLUTCHIO_REPO/crossplane-api/deploy/config-pkg-anynines.yaml"
    yq -i "with(select(document_index == 0); .spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\")" \
        "$KLUTCHIO_REPO/crossplane-api/deploy/provider-anynines.yaml"

    README_PATH="$KLUTCHIO_REPO/crossplane-api/README.md"
    sed -E 's|klutch/dataservices:v[0-9]+\.[0-9]+\.[0-9]+[^"]*|klutch/dataservices:'"${VERSION_NUMBER}"'|g' \
        "$README_PATH" >"${README_PATH}.tmp"
    mv "${README_PATH}.tmp" "$README_PATH"

    yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\"" \
        "$KLUTCHIO_REPO/provider-anynines/examples/provider/provider.yaml"
    yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\"" \
        "$KLUTCHIO_REPO/test/e2e/provider/manifests/install/provider.yaml"
    yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/dataservices:$VERSION_NUMBER\"" \
        "$KLUTCHIO_REPO/test/e2e/provider/manifests/configuration.yaml"
    log_normal "Klutchio manifest and README.md version references updated to ${VERSION_NUMBER}.\n"
}

update_documentation() {
    log_normal "Updating version references in klutchio-website documentation repo"
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
}

update_changelog() {
    # The ":1" in the variable expansion instructs the shell to omit the first character (the leading "v") from the version number when inserting it into the changelog.
    sed -E 's/^## Unreleased$/## '"[${VERSION_NUMBER:1}] - $(date "+%Y-%m-%d")/g" \
        "$KLUTCHIO_REPO/CHANGELOG.md" >"$KLUTCHIO_REPO/CHANGELOG.md.tmp"
    mv "$KLUTCHIO_REPO/CHANGELOG.md.tmp" "$KLUTCHIO_REPO/CHANGELOG.md"
}

commit_and_push_changes() {
    git add --all
    git commit -m "$@"
    git push "$GIT_REMOTE_NAME" "$GIT_BRANCH"
}

recreate_changelog_unreleased_section() {
    echo -e "# CHANGELOG\n\n## Unreleased\n$(tail -n +2 "$KLUTCHIO_REPO/CHANGELOG.md")" >"$KLUTCHIO_REPO/CHANGELOG.md"
    commit_and_push_changes "Recreate Changelog Section for unreleased Changes"
}

################################
#                              #
# SCRIPT EXECUTION STARTS HERE #
#                              #
################################

trap 'git checkout "$CURRENT_BRANCH"' EXIT
init "$@"
# build_docker_images
setup_git_branch
update_manifests
update_documentation
update_changelog
commit_and_push_changes "Prepare version $VERSION_NUMBER"
recreate_changelog_unreleased_section
