# Basic servicehandler app

In this application we are creating a very basic applications with only one entity: `User`. The app exposes endpoints for creating a user and fetching existing users by ID.

The goal is to show how we don't need to write any HTTP handlers for validation, encoding/decoding, and calling our service. We can focus on our application logic and let Go do the rest.
