// public/app.js

const API_BASE = ""; // Local relative URL works since we are serving this from the same host
let currentToken = localStorage.getItem("todo_api_token") || "";
let currentUser = null;
try {
    currentUser = JSON.parse(localStorage.getItem("todo_api_user")) || null;
} catch (e) {
    currentUser = null;
}

// Active filters state
let activeStatus = "";
let activePriority = "";
let searchTimeout = null;
let currentTodos = [];
let selectedTodoId = null;

// DOM Elements
const authScreen = document.getElementById("auth-screen");
const dashboardScreen = document.getElementById("dashboard-screen");
const loginForm = document.getElementById("login-form");
const registerForm = document.getElementById("register-form");
const toRegisterLink = document.getElementById("to-register");
const toLoginLink = document.getElementById("to-login");
const authSubtitle = document.getElementById("auth-subtitle");
const displayUsername = document.getElementById("display-username");
const logoutBtn = document.getElementById("logout-btn");
const searchInput = document.getElementById("search-input");
const priorityFilter = document.getElementById("priority-filter");
const todoList = document.getElementById("todo-list");

// Stats DOM
const statTotal = document.getElementById("stat-total");
const statPending = document.getElementById("stat-pending");
const statCompleted = document.getElementById("stat-completed");
const statCards = document.querySelectorAll(".stat-card");

// Modal DOM
const todoModal = document.getElementById("todo-modal");
const addTodoFab = document.getElementById("add-todo-fab");
const cancelTodoBtn = document.getElementById("cancel-todo-btn");
const saveTodoBtn = document.getElementById("save-todo-btn");
const todoForm = document.getElementById("todo-form");
const modalTitle = document.getElementById("modal-title");
const editTodoId = document.getElementById("edit-todo-id");
const todoTitleInput = document.getElementById("todo-title");
const todoDescInput = document.getElementById("todo-desc");
const todoDueDateInput = document.getElementById("todo-duedate");
const todoTagsInput = document.getElementById("todo-tags");

// ── UTILITIES ──

// Show custom toast message
function showToast(message, isSuccess = true) {
    const toast = document.getElementById("toast");
    toast.innerText = message;
    toast.style.borderLeft = `4px solid ${isSuccess ? 'var(--ios-green)' : 'var(--ios-red)'}`;
    toast.classList.add("show");
    setTimeout(() => {
        toast.classList.remove("show");
    }, 3000);
}

// Fetch helper that automatically adds JWT header
async function request(endpoint, options = {}) {
    const url = `${API_BASE}${endpoint}`;
    
    // Add auth header if we have a token
    const headers = {
        "Content-Type": "application/json",
        ...(options.headers || {})
    };
    
    if (currentToken) {
        headers["Authorization"] = `Bearer ${currentToken}`;
    }
    
    const config = {
        ...options,
        headers
    };
    
    try {
        const response = await fetch(url, config);
        
        // Handle unauthorized token (expired/invalid)
        if (response.status === 401) {
            handleLogout();
            showToast("Sesi habis, silakan login kembali.", false);
            return null;
        }
        
        const data = await response.json();
        if (!response.ok) {
            throw new Error(data.message || data.error || "Terjadi kesalahan server");
        }
        
        return data;
    } catch (err) {
        showToast(err.message, false);
        console.error("API Request Error:", err);
        throw err;
    }
}

// ── AUTHENTICATION ──

function checkAuth() {
    if (currentToken && currentUser) {
        authScreen.classList.remove("active");
        dashboardScreen.classList.add("active");
        displayUsername.innerText = currentUser.username;
        document.getElementById("user-avatar").innerText = currentUser.username.charAt(0).toUpperCase();
        loadTodos();
    } else {
        dashboardScreen.classList.remove("active");
        authScreen.classList.add("active");
    }
}

// Switch between Register and Login forms
toRegisterLink.addEventListener("click", (e) => {
    e.preventDefault();
    loginForm.classList.remove("active");
    registerForm.classList.add("active");
    authSubtitle.innerText = "Daftar untuk membuat akun baru";
});

toLoginLink.addEventListener("click", (e) => {
    e.preventDefault();
    registerForm.classList.remove("active");
    loginForm.classList.add("active");
    authSubtitle.innerText = "Masuk untuk melihat daftar tugas Anda";
});

// Handle Login Form Submit
loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const email = document.getElementById("login-email").value.trim();
    const password = document.getElementById("login-password").value;
    
    try {
        const res = await request("/login", {
            method: "POST",
            body: JSON.stringify({ email, password })
        });
        
        if (res && res.success) {
            currentToken = res.data.token;
            currentUser = res.data.user;
            localStorage.setItem("todo_api_token", currentToken);
            localStorage.setItem("todo_api_user", JSON.stringify(currentUser));
            showToast("Login Berhasil! Selamat datang.");
            checkAuth();
            loginForm.reset();
        }
    } catch (err) {
        // Error already handled by toast in request helper
    }
});

// Handle Register Form Submit
registerForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const username = document.getElementById("reg-username").value.trim();
    const email = document.getElementById("reg-email").value.trim();
    const password = document.getElementById("reg-password").value;
    
    try {
        const res = await request("/register", {
            method: "POST",
            body: JSON.stringify({ username, email, password })
        });
        
        if (res && res.success) {
            showToast("Pendaftaran Berhasil! Silakan masuk.");
            // Switch to login
            registerForm.reset();
            toLoginLink.click();
        }
    } catch (err) {
        // Error handled
    }
});

// Handle Logout
function handleLogout() {
    currentToken = "";
    currentUser = null;
    localStorage.removeItem("todo_api_token");
    localStorage.removeItem("todo_api_user");
    checkAuth();
}

logoutBtn.addEventListener("click", handleLogout);

// ── TODO ACTIONS & RENDERING ──

// Load Todos from API
async function loadTodos() {
    let queryParams = [];
    if (activeStatus) queryParams.push(`status=${activeStatus}`);
    if (activePriority) queryParams.push(`priority=${activePriority}`);
    
    const searchVal = searchInput.value.trim();
    if (searchVal) {
        queryParams.push(`search=${encodeURIComponent(searchVal)}`);
    }
    
    const queryString = queryParams.length ? `?${queryParams.join("&")}` : "";
    
    try {
        const res = await request(`/todos${queryString}`);
        if (res && res.success) {
            currentTodos = res.data.todos || [];
            renderTodoList(currentTodos);
            updateStats(res.data.stats);
            
            // Sync active selection
            if (selectedTodoId) {
                const stillExists = currentTodos.some(t => t.id === selectedTodoId);
                if (stillExists) {
                    selectTodo(selectedTodoId);
                } else {
                    clearSelection();
                }
            } else {
                clearSelection();
            }
        }
    } catch (err) {
        todoList.innerHTML = `<div class="empty-state"><p class="text-danger">Gagal mengambil daftar tugas.</p></div>`;
    }
}

// Render Todos into List Container
function renderTodoList(todos) {
    if (!todos || todos.length === 0) {
        todoList.innerHTML = `
            <div class="empty-state">
                <p>Tidak ada tugas yang ditemukan.</p>
            </div>
        `;
        return;
    }
    
    todoList.innerHTML = "";
    
    todos.forEach(todo => {
        const todoEl = document.createElement("div");
        todoEl.className = `todo-item ${todo.status === 'done' ? 'done' : ''} ${todo.status === 'in_progress' ? 'in-progress' : ''}`;
        todoEl.dataset.id = todo.id;
        
        // Due date rendering with overdue check
        let dueDateHTML = "";
        if (todo.due_date) {
            const dateObj = new Date(todo.due_date);
            const isOverdue = dateObj < new Date() && todo.status !== "done";
            const formattedDate = dateObj.toLocaleDateString("id-ID", {
                day: "numeric",
                month: "short",
                year: "numeric",
                hour: "2-digit",
                minute: "2-digit"
            });
            dueDateHTML = `
                <div class="due-date ${isOverdue ? 'overdue' : ''}">
                    📅 ${isOverdue ? 'Terlambat: ' : ''}${formattedDate}
                </div>
            `;
        }
        
        // Tags rendering
        let tagsHTML = "";
        if (todo.tags && todo.tags.length > 0) {
            tagsHTML = todo.tags.map(tag => `<span class="tag-badge">${tag}</span>`).join("");
        }
        
        // Priority rendering
        const priorityLabels = { low: "Rendah", medium: "Sedang", high: "Tinggi" };
        const priorityClass = todo.priority || "medium";
        const priorityHTML = `<span class="priority-pill ${priorityClass}">${priorityLabels[priorityClass]}</span>`;
        
        // Progress control button (Play/Pause)
        let progressButtonHTML = "";
        if (todo.status === "pending") {
            progressButtonHTML = `
                <button class="btn-icon play" onclick="toggleProgress('${todo.id}', '${todo.status}')" title="Mulai Proses">
                    <!-- iOS Play/Start Icon -->
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
                </button>
            `;
        } else if (todo.status === "in_progress") {
            progressButtonHTML = `
                <button class="btn-icon pause" onclick="toggleProgress('${todo.id}', '${todo.status}')" title="Tunda Tugas">
                    <!-- iOS Pause/Hold Icon -->
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="6" y="4" width="4" height="16"></rect><rect x="14" y="4" width="4" height="16"></rect></svg>
                </button>
            `;
        }

        todoEl.innerHTML = `
            <div class="todo-check">
                <div class="checkbox-circle ${todo.status === 'done' ? 'checked' : ''}" onclick="toggleComplete('${todo.id}', '${todo.status}')"></div>
            </div>
            <div class="todo-body">
                <div class="todo-title-row">
                    <h3 class="todo-item-title">${escapeHTML(todo.title)}</h3>
                    <div class="todo-actions">
                        ${progressButtonHTML}
                        <button class="btn-icon" onclick="openEditModal('${todo.id}')" title="Edit">
                            <!-- Edit Pencil Icon -->
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 1 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                        </button>
                        <button class="btn-icon delete" onclick="deleteTodo('${todo.id}')" title="Hapus">
                            <!-- Trash Icon -->
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path><line x1="10" y1="11" x2="10" y2="17"></line><line x1="14" y1="11" x2="14" y2="17"></line></svg>
                        </button>
                    </div>
                </div>
                ${todo.description ? `<p class="todo-item-desc">${escapeHTML(todo.description)}</p>` : ""}
                <div class="todo-meta">
                    ${priorityHTML}
                    ${dueDateHTML}
                    ${tagsHTML}
                </div>
            </div>
        `;
        
        // Register click handler for selection (iPad/Desktop) or modal opening (Mobile)
        todoEl.addEventListener("click", (e) => {
            if (e.target.closest("button") || e.target.closest(".checkbox-circle")) {
                return;
            }
            if (window.innerWidth >= 768) {
                selectTodo(todo.id);
            } else {
                openEditModal(todo.id);
            }
        });
        
        todoList.appendChild(todoEl);
    });
}

// Update UI stats indicators
function updateStats(stats) {
    if (!stats) return;
    
    statTotal.innerText = stats.total || 0;
    
    // "Sedang Jalan" = pending + in_progress
    const pendingCount = (stats.pending || 0) + (stats.in_progress || 0);
    statPending.innerText = pendingCount;
    statCompleted.innerText = stats.completed || 0;
}

// Toggle Todo Completed via API
async function toggleComplete(id, currentStatus) {
    const isDone = currentStatus === "done";
    
    try {
        if (!isDone) {
            // Use PATCH endpoint
            const res = await request(`/todos/${id}/complete`, { method: "PATCH" });
            if (res && res.success) {
                showToast("Tugas ditandai selesai!");
                loadTodos();
            }
        } else {
            // Revert back to pending using PUT update endpoint
            const res = await request(`/todos/${id}`, {
                method: "PUT",
                body: JSON.stringify({ status: "pending" })
            });
            if (res && res.success) {
                showToast("Tugas dikembalikan ke proses.");
                loadTodos();
            }
        }
    } catch (e) {
        // handled
    }
}

// Toggle Todo Progress (pending <=> in_progress) via API
async function toggleProgress(id, currentStatus) {
    const newStatus = currentStatus === "pending" ? "in_progress" : "pending";
    try {
        const res = await request(`/todos/${id}`, {
            method: "PUT",
            body: JSON.stringify({ status: newStatus })
        });
        if (res && res.success) {
            showToast(newStatus === "in_progress" ? "Tugas mulai diproses!" : "Tugas ditunda.");
            loadTodos();
        }
    } catch (e) {
        // handled
    }
}

// Delete Todo via API
async function deleteTodo(id) {
    if (!confirm("Apakah Anda yakin ingin menghapus tugas ini?")) return;
    
    try {
        const res = await request(`/todos/${id}`, { method: "DELETE" });
        if (res && res.success) {
            showToast("Tugas berhasil dihapus.");
            loadTodos();
        }
    } catch (e) {
        // handled
    }
}

// ── MODAL SHEET & FORMS ──

function openCreateModal() {
    modalTitle.innerText = "Tugas Baru";
    editTodoId.value = "";
    todoForm.reset();
    
    // Pre-populate with current local date/time
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    tomorrow.setHours(9, 0, 0, 0);
    // Format to yyyy-MM-ddThh:mm
    const year = tomorrow.getFullYear();
    const month = String(tomorrow.getMonth() + 1).padStart(2, '0');
    const day = String(tomorrow.getDate()).padStart(2, '0');
    const hours = String(tomorrow.getHours()).padStart(2, '0');
    const minutes = String(tomorrow.getMinutes()).padStart(2, '0');
    todoDueDateInput.value = `${year}-${month}-${day}T${hours}:${minutes}`;

    todoModal.classList.add("active");
}

async function openEditModal(id) {
    try {
        const res = await request(`/todos/${id}`);
        if (res && res.success) {
            const todo = res.data;
            modalTitle.innerText = "Edit Tugas";
            editTodoId.value = todo.id;
            todoTitleInput.value = todo.title;
            todoDescInput.value = todo.description || "";
            
            // Set priority radio
            const priorityRadio = document.querySelector(`input[name="todo-priority"][value="${todo.priority}"]`);
            if (priorityRadio) priorityRadio.checked = true;
            
            // Set due date format (yyyy-MM-ddThh:mm)
            if (todo.due_date) {
                const d = new Date(todo.due_date);
                // Adjust for local timezone input
                const offset = d.getTimezoneOffset() * 60000;
                const localISOTime = (new Date(d.getTime() - offset)).toISOString().slice(0, 16);
                todoDueDateInput.value = localISOTime;
            } else {
                todoDueDateInput.value = "";
            }
            
            // Set tags
            todoTagsInput.value = todo.tags ? todo.tags.join(", ") : "";
            
            todoModal.classList.add("active");
        }
    } catch (e) {
        // handled
    }
}

function closeModal() {
    todoModal.classList.remove("active");
}

// Save Todo (Create or Update)
todoForm.addEventListener("submit", async (e) => {
    e.preventDefault();
});

saveTodoBtn.addEventListener("click", async () => {
    const title = todoTitleInput.value.trim();
    if (!title) {
        showToast("Judul tidak boleh kosong.", false);
        return;
    }
    
    const description = todoDescInput.value.trim();
    const priority = document.querySelector('input[name="todo-priority"]:checked').value;
    
    // Parse tags: split by comma, trim spaces, remove empty tags
    const tags = todoTagsInput.value
        .split(",")
        .map(t => t.trim())
        .filter(t => t.length > 0);
        
    // Format due date to RFC3339
    let due_date = "";
    if (todoDueDateInput.value) {
        const localDate = new Date(todoDueDateInput.value);
        due_date = localDate.toISOString(); // converts to UTC ISO-8601 which net/http handles
    }
    
    const id = editTodoId.value;
    const isEdit = !!id;
    
    const url = isEdit ? `/todos/${id}` : "/todos";
    const method = isEdit ? "PUT" : "POST";
    
    const payload = {
        title,
        description,
        priority,
        due_date,
        tags
    };
    

    // Actually in JS we just send normal properties, Go's json.Unmarshal maps to pointers.
    // If due_date is empty, send nil/null so it is ignored or cleared.
    const finalPayload = { ...payload };
    if (!due_date) {
        if (isEdit) {
            finalPayload.due_date = null;
        } else {
            delete finalPayload.due_date;
        }
    }
    
    try {
        const res = await request(url, {
            method,
            body: JSON.stringify(finalPayload)
        });
        
        if (res && res.success) {
            showToast(isEdit ? "Tugas berhasil diperbarui!" : "Tugas berhasil ditambahkan!");
            closeModal();
            loadTodos();
        }
    } catch (e) {
        // handled
    }
});

// ── EVENTS ──

addTodoFab.addEventListener("click", openCreateModal);
cancelTodoBtn.addEventListener("click", closeModal);

// Segmented control click handlers
const segments = document.querySelectorAll(".segment");
segments.forEach(segment => {
    segment.addEventListener("click", (e) => {
        segments.forEach(s => s.classList.remove("active"));
        segment.classList.add("active");
        
        activeStatus = segment.dataset.status;
        loadTodos();
    });
});

// Priority filter select handler
priorityFilter.addEventListener("change", (e) => {
    activePriority = e.target.value;
    loadTodos();
});

// Search input handler with debounce
searchInput.addEventListener("input", () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
        loadTodos();
    }, 400); // 400ms debounce
});

// HTML escaping helper
function escapeHTML(str) {
    if (!str) return "";
    return str
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// Master-Detail Panel Functions
function selectTodo(id) {
    selectedTodoId = id;
    
    // Highlight active todo
    document.querySelectorAll(".todo-item").forEach(item => {
        if (item.dataset.id === id) {
            item.classList.add("active-todo");
        } else {
            item.classList.remove("active-todo");
        }
    });
    
    const todo = currentTodos.find(t => t.id === id);
    if (!todo) return;
    
    const emptyState = document.querySelector(".detail-empty-state");
    const detailContent = document.querySelector(".detail-content");
    
    if (emptyState) emptyState.style.display = "none";
    if (detailContent) detailContent.style.display = "flex";
    
    // Populate details
    document.getElementById("detail-title").innerText = todo.title;
    
    const statusMap = { pending: "Tunda", in_progress: "Proses", done: "Selesai" };
    const statusBadge = document.getElementById("detail-status");
    if (statusBadge) {
        statusBadge.innerText = statusMap[todo.status] || todo.status;
        statusBadge.className = `tag-badge ${todo.status === 'done' ? 'checked' : ''}`;
    }
    
    const priorityLabel = document.getElementById("detail-priority");
    if (priorityLabel) {
        const priorityLabels = { low: "Rendah", medium: "Sedang", high: "Tinggi" };
        priorityLabel.innerText = priorityLabels[todo.priority] || "Sedang";
        priorityLabel.className = `priority-pill ${todo.priority}`;
    }
    
    const dueDateSpan = document.getElementById("detail-duedate");
    if (dueDateSpan) {
        if (todo.due_date) {
            const dateObj = new Date(todo.due_date);
            dueDateSpan.innerText = dateObj.toLocaleDateString("id-ID", {
                day: "numeric",
                month: "long",
                year: "numeric",
                hour: "2-digit",
                minute: "2-digit"
            });
        } else {
            dueDateSpan.innerText = "Tidak ada";
        }
    }
    
    // Description Card
    const descCard = document.getElementById("detail-desc-card");
    const descText = document.getElementById("detail-desc");
    if (descCard && descText) {
        if (todo.description) {
            descText.innerText = todo.description;
            descCard.style.display = "block";
        } else {
            descCard.style.display = "none";
        }
    }
    
    // Tags Card
    const tagsCard = document.getElementById("detail-tags-card");
    const tagsContainer = document.getElementById("detail-tags");
    if (tagsCard && tagsContainer) {
        if (todo.tags && todo.tags.length > 0) {
            tagsContainer.innerHTML = todo.tags.map(tag => `<span class="tag-badge">${tag}</span>`).join("");
            tagsCard.style.display = "block";
        } else {
            tagsCard.style.display = "none";
        }
    }
}

function clearSelection() {
    selectedTodoId = null;
    document.querySelectorAll(".todo-item").forEach(item => item.classList.remove("active-todo"));
    
    const emptyState = document.querySelector(".detail-empty-state");
    const detailContent = document.querySelector(".detail-content");
    
    if (emptyState) emptyState.style.display = "flex";
    if (detailContent) detailContent.style.display = "none";
}

// Detail Panel Button Event Listeners
document.getElementById("detail-edit-btn").addEventListener("click", () => {
    if (selectedTodoId) openEditModal(selectedTodoId);
});

document.getElementById("detail-delete-btn").addEventListener("click", () => {
    if (selectedTodoId) {
        deleteTodo(selectedTodoId);
    }
});

// ── INIT ──
checkAuth();
