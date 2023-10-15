use axum::{
    extract::{Form, State},
    response::{Html, Json},
    routing::{get, post},
    Router,
};
use minijinja::{render, Environment};
use serde::{Deserialize, Serialize};
use std::{
    net::SocketAddr,
    sync::{Arc, RwLock},
};
use uuid::Uuid;

#[derive(Clone, Debug, Deserialize, Serialize)]
struct Todo {
    id: String,
    title: String,
    done: bool,
}

#[derive(Debug, Default, Serialize)]
struct AppState {
    todos: Vec<Todo>,
}

type SharedState = Arc<RwLock<AppState>>;

#[derive(Deserialize)]
struct CreateInput {
    title: String,
}

#[derive(Deserialize)]
struct UpdateInput {
    id: String,
    done: Option<String>,
}

const MAIN_TEMPLATE: &'static str = r###"
<!DOCTYPE html>
<html lang="en-US" dir="ltr">
<head>
    <title>Get things done</title>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1" />
    <style type="text/css">
    .form {
        display: flex;
        flex-direction: column;
    }
    .input {
        padding: 12px;
        font-size: 1.2em;
    }
    .todo-item {
        display: flex;
        flex-direction: row;
        font-family: sans-serif;
        font-size: 2.0em;
    }
    .done {
        color: #999;
        text-decoration: line-through;
    }
    input.check {
        transform: scale(2);
        margin: 12px;
    }
    .inline-form {
        display: flex;
        flex-direction: row;
    }
    ul {
        list-style-type: none;
        padding: 0;
    }
    </style>
    <link rel="icon" href="data:,">
    <script src="https://unpkg.com/htmx.org@latest"></script>
</head>
<body>
    <div class="content" id="content">
         {% include "content_template" %}
    </div>
</body>
</html>
"###;

const CONTENT_TEMPLATE: &'static str = r###"
        <form hx-boost="true" hx-post="/create" class="form" hx-target="#content">
            <input name="title" class="input" type="text" placeholder="What needs to be done?" autofocus>
        </form>
        {% if todos|length %}
        <ul>
            {% for todo in todos %}
            <li class="todo-item {% if todo.done %}done{% endif %}">
                <form hx-target="#content" class="inline-form">
                    <input name="id" type="hidden" value="{{ todo.id }}">
                    <input class="check" name="done" type="checkbox" hx-trigger="click" hx-post="/update" {% if todo.done %}checked{% endif %}>
                    {{ todo.title }}
                </form>
            </li>
            {% endfor %}
        </ul>
        {% endif %}
"###;

#[tokio::main]
async fn main() {
    let app = app();

    let addr = SocketAddr::from(([127, 0, 0, 1], 3000));

    axum::Server::bind(&addr)
        .serve(app.into_make_service())
        .await
        .unwrap();
}

fn app() -> Router {
    let state = SharedState::default();

    Router::new()
        .route("/", get(read))
        .route("/create", post(create))
        .route("/update", post(update))
        .route("/todos", get(todos))
        .with_state(Arc::clone(&state))
}

async fn todos(State(state): State<SharedState>) -> Json<Vec<Todo>> {
    let todos = &state.read().unwrap().todos;
    Json(todos.clone())
}

async fn read(State(state): State<SharedState>) -> Html<String> {
    let mut env = Environment::new();
    if env
        .add_template("content_template", CONTENT_TEMPLATE)
        .is_err()
    {
        panic!("could not add template")
    }
    let todos = &state.read().unwrap().todos;

    let html = render!(in env, MAIN_TEMPLATE, todos => todos);
    Html(html)
}

async fn create(State(state): State<SharedState>, input: Form<CreateInput>) -> Html<String> {
    let title = input.title.to_owned();
    let id = Uuid::new_v4().to_string();
    let todo = Todo {
        done: false,
        id,
        title,
    };

    state.write().unwrap().todos.push(todo);

    let todos = &state.read().unwrap().todos;
    let html = render!(CONTENT_TEMPLATE, todos => todos);
    Html(html)
}

async fn update(State(state): State<SharedState>, input: Form<UpdateInput>) -> Html<String> {
    let id = input.id.as_str();
    let done = match &input.done {
        Some(v) => v.to_owned(),
        None => String::from(""),
    };

    let mut writer = state.write().unwrap();
    for todo in writer.todos.iter_mut() {
        if todo.id == id {
            if done == "on" {
                todo.done = true;
            } else {
                todo.done = false;
            }
        }
    }
    drop(writer);

    let todos = &state.read().unwrap().todos;
    let html = render!(CONTENT_TEMPLATE, todos => todos);
    Html(html)
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::{
        body::Body,
        http::{self, Request, StatusCode},
    };
    use serde_json::json;
    use serde_json::Value;
    use tower::{Service, ServiceExt};

    #[tokio::test]
    async fn test_read_ok() {
        let response = app()
            .oneshot(
                Request::builder()
                    .method(http::Method::GET)
                    .uri("/")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();

        assert_eq!(response.status(), StatusCode::OK);
        let body = hyper::body::to_bytes(response.into_body()).await.unwrap();
        assert!(body.len() > 0);
    }

    #[tokio::test]
    async fn test_create_ok() {
        let mut app = app();
        let value = &[("title", "foo")];

        let request = Request::builder()
            .method(http::Method::POST)
            .header("content-type", "application/x-www-form-urlencoded")
            .uri("/create")
            .body(Body::from(serde_urlencoded::to_string(value).unwrap()))
            .unwrap();

        let response = ServiceExt::<Request<Body>>::ready(&mut app)
            .await
            .unwrap()
            .call(request)
            .await
            .unwrap();

        assert_eq!(response.status(), StatusCode::OK);
        let body = hyper::body::to_bytes(response.into_body()).await.unwrap();
        assert!(body.len() > 0);

        let request = Request::builder()
            .method(http::Method::GET)
            .uri("/todos")
            .body(Body::empty())
            .unwrap();

        let response = ServiceExt::<Request<Body>>::ready(&mut app)
            .await
            .unwrap()
            .call(request)
            .await
            .unwrap();

        assert_eq!(response.status(), StatusCode::OK);
        let body = hyper::body::to_bytes(response.into_body()).await.unwrap();

        let body: Value = serde_json::from_slice(&body).unwrap();
        let json = json!(body);
        let todos: Vec<Todo> = serde_json::from_value(json).unwrap();
        let todo = todos.get(0).unwrap();
        assert!(todo.id.len() > 0);

        let value = &[("done", "on"), ("id", &todo.id)];

        let request = Request::builder()
            .method(http::Method::POST)
            .header("content-type", "application/x-www-form-urlencoded")
            .uri("/update")
            .body(Body::from(serde_urlencoded::to_string(value).unwrap()))
            .unwrap();

        let response = ServiceExt::<Request<Body>>::ready(&mut app)
            .await
            .unwrap()
            .call(request)
            .await
            .unwrap();

        assert_eq!(response.status(), StatusCode::OK);
        let body = hyper::body::to_bytes(response.into_body()).await.unwrap();
        assert!(body.len() > 0);
    }
}
