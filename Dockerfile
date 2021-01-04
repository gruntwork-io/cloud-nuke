# To build this locally, just update the CIRCLE_TAG environment value and run this command: 
#    export CIRCLE_TAG=1.0.0 && DOCKER_BUILDKIT=1 docker build --compress --build-arg "VERSION=$CIRCLE_TAG" --tag "gruntwork-io/cloud-nuke:$CIRCLE_TAG" --tag "gruntwork-io/cloud-nuke:latest" .
# To run this: 
#    docker run -it --rm -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_SESSION_TOKEN gruntwork-io/cloud-nuke
# Linted with https://github.com/hadolint/hadolint:
#    docker run --rm -i hadolint/hadolint < Dockerfile
# And linted with https://github.com/replicatedhq/dockerfilelint
#    docker run --rm -v "$PWD/Dockerfile:/Dockerfile" replicated/dockerfilelint /Dockerfile


####### Builder ########
FROM golang:1.15-alpine as builder
#Preparing working directory and copying files
WORKDIR /wrk
COPY . . 

#You can overrride by setting the CIRCLE_TAG environment variable, 0.0.0 is just a, so you'll notice if you fot
ARG VERSION=0.0.0
RUN go build -o dest/cloud-nuke -ldflags="-X main.VERSION=$CIRCLE_TAG"

####### Runner ########
#Choosing alpine base as runner, since we don't need the go compiler to run the program
FROM alpine:3 as runner

#Set working directory
WORKDIR /wrk
#Create non-root user, to make the containersafer to run
RUN addgroup -S app && adduser -S -G app app
#Own the working folder
RUN chown app:app . 
#Switch to the new non-root user
USER app
#Copy over just the binary from the builder image and change the owner to the non-root user
COPY --from=builder --chown=app:app /wrk/dest/ .
#Make the binary executable
RUN chmod +x cloud-nuke
#Set the entrypoint so that cloud-nuke is the default action
ENTRYPOINT ["./cloud-nuke"]
