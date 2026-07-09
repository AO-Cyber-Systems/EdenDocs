# syntax=docker/dockerfile:1.4
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
#
# EdenDocs branded container image — from-source pattern (DIST-01), with a
# verify-caps capability-check stage ported from docker/from-packages (DIST-02).
# NO LABELs here: OCI labels/annotations are injected at CI level
# (docker/metadata-action) in .github/workflows/docker-publish.yml.
ARG base_image="ubuntu:24.04"

# ---- Stage 1: assemble — upstream from-source content, single filesystem ----
FROM ${base_image} AS assemble
# Engine run-time deps + adduser/findutils/openssl/cpio + tzdata + CJK fonts,
# borrowed verbatim from docker/from-source/Ubuntu (minus the interactive editor
# `nano`, which a headless container does not need). Delete apt lists (DL3009).
# hadolint ignore=DL3008,DL3009
RUN apt-get update && \
    apt-get -y install libpng16-16 fontconfig adduser cpio tzdata \
      findutils libcap2-bin openssl openssh-client \
      libxcb-shm0 libxcb-render0 libxrender1 libxext6 \
      fonts-wqy-zenhei fonts-wqy-microhei fonts-droid-fallback \
      fonts-noto-cjk ca-certificates && \
    rm -rf /var/lib/apt/lists/*
# The build context root is docker/eden/ which contains instdir/ (staged by
# scripts/eden/build-release.sh: usr/bin/cool* + opt/collaboraoffice + branding).
COPY instdir /
RUN setcap cap_fowner,cap_chown,cap_sys_chroot=ep /usr/bin/coolforkit-caps && \
    setcap cap_sys_admin=ep /usr/bin/coolmount && \
    adduser --quiet --system --group --home /opt/cool cool && \
    rm -rf /opt/cool && mkdir -p /opt/cool/child-roots /opt/cool/cache && \
    coolwsd-systemplate-setup /opt/cool/systemplate /opt/collaboraoffice >/dev/null 2>&1 && \
    touch /var/log/coolwsd.log && chown cool:cool /var/log/coolwsd.log && \
    chown -R cool:cool /opt/ && chown -R cool:cool /etc/coolwsd && chmod g+w /etc/passwd

# ---- Stage 2: verify-caps — exercise COPY --from and assert caps survived ----
# Copying the two capability-bearing binaries out of `assemble` exercises the
# exact same COPY --from mechanism the runtime stage uses; getcap is already
# present here (libcap2-bin). Fail the build rather than ship a jail that cannot
# start because the security.capability xattr was dropped on COPY --from.
FROM assemble AS verify-caps
COPY --from=assemble /usr/bin/coolforkit-caps /usr/bin/coolmount /probe/
RUN set -eu; \
    fc="$(/sbin/getcap /probe/coolforkit-caps)"; mc="$(/sbin/getcap /probe/coolmount)"; \
    echo "post-COPY caps -- coolforkit-caps: ${fc:-<none>} | coolmount: ${mc:-<none>}"; \
    case "$fc" in *cap_sys_chroot*) ;; *) echo "FATAL: cap_sys_chroot lost on COPY --from" >&2; exit 1;; esac; \
    case "$mc" in *cap_sys_admin*) ;; *) echo "FATAL: cap_sys_admin lost on COPY --from" >&2; exit 1;; esac; \
    touch /caps-verified

# ---- Stage 3: runtime ----
FROM ${base_image}
COPY --from=assemble / /
# Force the verify-caps gate into every build: this pulls in stage 2, which
# fails the build if the security.capability xattr did not survive COPY --from.
COPY --from=verify-caps /caps-verified /opt/cool/.caps-verified
EXPOSE 9980
USER 100
HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD ["/usr/bin/coolwsd", "--probe"]
ENTRYPOINT ["/usr/bin/coolwsd", "--use-env-vars", \
    "--o:sys_template_path=/opt/cool/systemplate", \
    "--o:child_root_path=/opt/cool/child-roots", \
    "--o:file_server_root_path=/usr/share/coolwsd", \
    "--o:cache_files.path=/opt/cool/cache", \
    "--o:logging.color=false", "--o:stop_on_config_change=true"]
