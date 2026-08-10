FROM eclipse-temurin:21-jre-jammy

ARG GRAPHHOPPER_VERSION=11.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && curl --fail --location --silent --show-error \
      "https://repo1.maven.org/maven2/com/graphhopper/graphhopper-web/${GRAPHHOPPER_VERSION}/graphhopper-web-${GRAPHHOPPER_VERSION}.jar" \
      --output /app.jar \
    && curl --fail --location --silent --show-error \
      "https://repo1.maven.org/maven2/com/graphhopper/graphhopper-web/${GRAPHHOPPER_VERSION}/graphhopper-web-${GRAPHHOPPER_VERSION}.jar.sha1" \
      --output /app.jar.sha1 \
    && printf '%s  %s\n' "$(cat /app.jar.sha1)" /app.jar | sha1sum --check --strict \
    && rm /app.jar.sha1 \
    && rm -rf /var/lib/apt/lists/*

COPY config.yml /config.yml

EXPOSE 8989

ENTRYPOINT ["java", "-Xms128m", "-Xmx1g", "-jar", "/app.jar"]
CMD ["server", "/config.yml"]
