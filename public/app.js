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
let activeTag = "";
let activeSortBy = "created_at";
let activeSortOrder = "desc";
let searchTimeout = null;
let currentTodos = [];
let selectedTodoId = null;

// Selection Mode State
let isSelectionMode = false;
let selectedTodos = new Set();

// Pagination State
let currentPage = 1;
const limitPerPage = 10;
let hasMoreTodos = false;
let isLoadingMore = false;

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
const tagFilter = document.getElementById("tag-filter");
const sortFilter = document.getElementById("sort-filter");
const todoList = document.getElementById("todo-list");
const selectModeBtn = document.getElementById("select-mode-btn");
const bulkActionBar = document.getElementById("bulk-action-bar");
const bulkSelectCount = document.getElementById("bulk-select-count");
const bulkCompleteBtn = document.getElementById("bulk-complete-btn");
const bulkDeleteBtn = document.getElementById("bulk-delete-btn");

// Theme Toggle elements
const themeToggleBtn = document.getElementById("theme-toggle-btn");
const sunIcon = themeToggleBtn ? themeToggleBtn.querySelector(".sun-icon") : null;
const moonIcon = themeToggleBtn ? themeToggleBtn.querySelector(".moon-icon") : null;

// Infinite Scroll elements
const scrollSentinel = document.getElementById("scroll-sentinel");
const loadingSpinner = document.getElementById("loading-spinner");

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
const modalSubtasksContainer = document.getElementById("modal-subtasks-container");
const addSubtaskBtn = document.getElementById("add-subtask-btn");
const detailSubtasksCard = document.getElementById("detail-subtasks-card");
const detailSubtasksContainer = document.getElementById("detail-subtasks-container");
let modalSubtasks = [];

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

// Format remaining time or overdue duration in Indonesian
function formatRemainingTime(dueDateStr, status) {
    if (!dueDateStr) return "";
    
    const dueDate = new Date(dueDateStr);
    const now = new Date();
    
    const isDone = status === "done";
    const diffMs = dueDate - now;
    const isOverdue = diffMs < 0;
    
    // If completed, we don't display remaining or overdue info
    if (isDone) return "";
    
    const absDiffMs = Math.abs(diffMs);
    const diffMins = Math.floor(absDiffMs / (1000 * 60));
    const diffHours = Math.floor(absDiffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(absDiffMs / (1000 * 60 * 60 * 24));
    
    let timeText = "";
    if (diffDays > 0) {
        timeText = `${diffDays} hari`;
    } else if (diffHours > 0) {
        const minsRemaining = diffMins % 60;
        timeText = `${diffHours} jam${minsRemaining > 0 ? ` ${minsRemaining} menit` : ""}`;
    } else {
        timeText = `${diffMins} menit`;
    }
    
    return isOverdue ? `Terlambat ${timeText}` : `Sisa ${timeText}`;
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
// Load Todos from API (Halaman 1)
async function loadTodos() {
    currentPage = 1;
    isLoadingMore = false;
    
    let queryParams = [];
    if (activeStatus) queryParams.push(`status=${activeStatus}`);
    if (activePriority) queryParams.push(`priority=${activePriority}`);
    if (activeTag) queryParams.push(`tag=${encodeURIComponent(activeTag)}`);
    if (activeSortBy) queryParams.push(`sort_by=${activeSortBy}`);
    if (activeSortOrder) queryParams.push(`order=${activeSortOrder}`);
    
    const searchVal = searchInput.value.trim();
    if (searchVal) {
        queryParams.push(`search=${encodeURIComponent(searchVal)}`);
    }
    
    // Kirim parameter paginasi
    queryParams.push(`page=${currentPage}`);
    queryParams.push(`limit=${limitPerPage}`);
    
    const queryString = queryParams.length ? `?${queryParams.join("&")}` : "";
    
    try {
        const res = await request(`/todos${queryString}`);
        if (res && res.success) {
            currentTodos = res.data.todos || [];
            
            // Render ulang dari awal
            renderTodoList(currentTodos);
            updateStats(res.data.stats);
            
            // Sinkronisasi status has_more
            const pag = res.data.pagination;
            hasMoreTodos = pag ? pag.has_more : false;
            
            if (scrollSentinel) {
                scrollSentinel.style.display = hasMoreTodos ? "flex" : "none";
            }
            if (loadingSpinner) {
                loadingSpinner.style.display = "none";
            }
            
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

// Load More Todos dari API (Halaman > 1)
async function loadMoreTodos() {
    if (isLoadingMore || !hasMoreTodos) return;
    
    isLoadingMore = true;
    if (loadingSpinner) {
        loadingSpinner.style.display = "block";
    }
    
    currentPage++;
    
    let queryParams = [];
    if (activeStatus) queryParams.push(`status=${activeStatus}`);
    if (activePriority) queryParams.push(`priority=${activePriority}`);
    if (activeTag) queryParams.push(`tag=${encodeURIComponent(activeTag)}`);
    if (activeSortBy) queryParams.push(`sort_by=${activeSortBy}`);
    if (activeSortOrder) queryParams.push(`order=${activeSortOrder}`);
    
    const searchVal = searchInput.value.trim();
    if (searchVal) {
        queryParams.push(`search=${encodeURIComponent(searchVal)}`);
    }
    
    queryParams.push(`page=${currentPage}`);
    queryParams.push(`limit=${limitPerPage}`);
    
    const queryString = queryParams.length ? `?${queryParams.join("&")}` : "";
    
    try {
        const res = await request(`/todos${queryString}`);
        if (res && res.success) {
            const newTodos = res.data.todos || [];
            currentTodos = currentTodos.concat(newTodos);
            
            // Tambahkan item baru ke daftar yang sudah ada
            appendTodoList(newTodos);
            updateStats(res.data.stats);
            
            const pag = res.data.pagination;
            hasMoreTodos = pag ? pag.has_more : false;
            
            if (scrollSentinel) {
                scrollSentinel.style.display = hasMoreTodos ? "flex" : "none";
            }
        } else {
            currentPage--;
        }
    } catch (err) {
        currentPage--;
    } finally {
        isLoadingMore = false;
        if (loadingSpinner) {
            loadingSpinner.style.display = "none";
        }
    }
}

// Helper untuk membuat element kartu Todo DOM tunggal
function createTodoElement(todo, searchQuery) {
    const isSelected = selectedTodos.has(todo.id);
    const todoEl = document.createElement("div");
    todoEl.className = `todo-item ${todo.status === 'done' ? 'done' : ''} ${todo.status === 'in_progress' ? 'in-progress' : ''} ${isSelected ? 'selected-item' : ''}`;
    todoEl.dataset.id = todo.id;
    
    // Due date rendering
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
        const remainingText = formatRemainingTime(todo.due_date, todo.status);
        const displayRemaining = remainingText ? ` (${remainingText})` : "";
        
        dueDateHTML = `
            <div class="due-date ${isOverdue ? 'overdue' : ''}">
                📅 ${formattedDate}${displayRemaining}
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
    
    // Progress badge rendering
    let subTaskProgressHTML = "";
    let subTaskProgressBarHTML = "";
    if (todo.sub_tasks && todo.sub_tasks.length > 0) {
        const totalSub = todo.sub_tasks.length;
        const completedSub = todo.sub_tasks.filter(st => st.status === 'done').length;
        const pct = Math.round((completedSub / totalSub) * 100);
        const isComplete = pct === 100;
        subTaskProgressHTML = `<span class="subtask-progress-badge">☑️ ${completedSub}/${totalSub}</span>`;
        subTaskProgressBarHTML = `
            <div class="subtask-progress-container">
                <div class="subtask-progress-track">
                    <div class="subtask-progress-fill ${isComplete ? 'complete' : ''}" style="width: ${pct}%"></div>
                </div>
                <span class="subtask-progress-label">${pct}%</span>
            </div>
        `;
    }
    
    // Progress button
    let progressButtonHTML = "";
    if (todo.status === "pending") {
        progressButtonHTML = `
            <button class="btn-icon play" onclick="toggleProgress('${todo.id}', '${todo.status}')" title="Mulai Proses">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
            </button>
        `;
    } else if (todo.status === "in_progress") {
        progressButtonHTML = `
            <button class="btn-icon pause" onclick="toggleProgress('${todo.id}', '${todo.status}')" title="Tunda Tugas">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="6" y="4" width="4" height="16"></rect><rect x="14" y="4" width="4" height="16"></rect></svg>
            </button>
        `;
    }

    todoEl.innerHTML = `
        <div class="todo-check">
            <div class="checkbox-circle ${todo.status === 'done' ? 'checked' : ''}" onclick="handleCheckboxClick(event, '${todo.id}', '${todo.status}')"></div>
        </div>
        <div class="todo-body">
            <div class="todo-title-row">
                <h3 class="todo-item-title">${highlightText(todo.title, searchQuery)}</h3>
                <div class="todo-actions">
                    ${progressButtonHTML}
                    <button class="btn-icon" onclick="openEditModal('${todo.id}')" title="Edit">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 1 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                    </button>
                    <button class="btn-icon delete" onclick="deleteTodo('${todo.id}')" title="Hapus">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path><line x1="10" y1="11" x2="10" y2="17"></line><line x1="14" y1="11" x2="14" y2="17"></line></svg>
                    </button>
                </div>
            </div>
            ${todo.description ? `<p class="todo-item-desc">${highlightText(todo.description, searchQuery)}</p>` : ""}
            ${subTaskProgressBarHTML}
            <div class="todo-meta">
                ${priorityHTML}
                ${subTaskProgressHTML}
                ${dueDateHTML}
                ${tagsHTML}
            </div>
        </div>
    `;
    
    todoEl.addEventListener("click", (e) => {
        if (e.target.closest("button")) {
            return;
        }
        if (isSelectionMode) {
            toggleTodoSelection(todo.id);
            return;
        }
        if (e.target.closest(".checkbox-circle")) {
            return;
        }
        if (window.innerWidth >= 768) {
            selectTodo(todo.id);
        } else {
            openEditModal(todo.id);
        }
    });
    
    return todoEl;
}

// Render list todo dari awal
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
    appendTodoList(todos);
}

// Tambahkan item todo baru ke daftar yang sudah ada
function appendTodoList(todos) {
    if (!todos) return;
    const searchQuery = searchInput ? searchInput.value.trim() : "";
    
    todos.forEach(todo => {
        const todoEl = createTodoElement(todo, searchQuery);
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

    // Populate dynamic tag filter select options
    if (stats.tags) {
        const previousSelection = activeTag;
        
        tagFilter.innerHTML = '<option value="">Semua Tag</option>';
        stats.tags.forEach(tag => {
            const opt = document.createElement("option");
            opt.value = tag;
            opt.innerText = tag;
            if (tag === previousSelection) {
                opt.selected = true;
            }
            tagFilter.appendChild(opt);
        });
    }
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

// Toggle Sub-task Completion status in Detail Panel with Auto-save to server
async function toggleSubTaskStatus(todoId, subTaskId) {
    const todo = currentTodos.find(t => t.id === todoId);
    if (!todo) return;
    
    const subTasks = todo.sub_tasks.map(st => {
        if (st.id === subTaskId) {
            return {
                ...st,
                status: st.status === 'done' ? 'pending' : 'done'
            };
        }
        return st;
    });
    
    try {
        const res = await request(`/todos/${todoId}`, {
            method: "PUT",
            body: JSON.stringify({ sub_tasks: subTasks })
        });
        if (res && res.success) {
            // Update the local todo object
            const index = currentTodos.findIndex(t => t.id === todoId);
            if (index !== -1) {
                currentTodos[index] = res.data;
            }
            renderTodoList(currentTodos);
            selectTodo(todoId);
        }
    } catch (e) {
        showToast("Gagal memperbarui status sub-tugas.", false);
    }
}

// Render sub-tasks inside Edit/Create Modal
function renderModalSubtasks() {
    if (!modalSubtasksContainer) return;
    modalSubtasksContainer.innerHTML = "";
    
    modalSubtasks.forEach((st, index) => {
        const itemEl = document.createElement("div");
        itemEl.className = "modal-subtask-item";
        itemEl.innerHTML = `
            <input type="text" placeholder="Nama sub-tugas..." value="${escapeHTML(st.title)}">
            <button type="button" class="btn-remove-subtask" title="Hapus Sub-tugas">&times;</button>
        `;
        
        // Update model title on input
        const input = itemEl.querySelector("input");
        input.addEventListener("input", (e) => {
            modalSubtasks[index].title = e.target.value;
        });
        
        // Remove button handler
        const removeBtn = itemEl.querySelector(".btn-remove-subtask");
        removeBtn.addEventListener("click", () => {
            // Read current titles first to preserve any typing
            const inputs = modalSubtasksContainer.querySelectorAll(".modal-subtask-item input");
            inputs.forEach((inp, idx) => {
                if (modalSubtasks[idx]) {
                    modalSubtasks[idx].title = inp.value;
                }
            });
            modalSubtasks.splice(index, 1);
            renderModalSubtasks();
        });
        
        modalSubtasksContainer.appendChild(itemEl);
    });
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

    modalSubtasks = [];
    renderModalSubtasks();

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
            
            modalSubtasks = todo.sub_tasks ? JSON.parse(JSON.stringify(todo.sub_tasks)) : [];
            renderModalSubtasks();
            
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
    
    // Update subtasks title from inputs before saving
    if (modalSubtasksContainer) {
        const subTaskInputs = modalSubtasksContainer.querySelectorAll(".modal-subtask-item input");
        subTaskInputs.forEach((input, index) => {
            if (modalSubtasks[index]) {
                modalSubtasks[index].title = input.value.trim();
            }
        });
    }
    
    // Filter out sub-tasks with empty titles
    const subtasksToSend = modalSubtasks.filter(st => st.title.trim().length > 0);
    
    const id = editTodoId.value;
    const isEdit = !!id;
    
    const url = isEdit ? `/todos/${id}` : "/todos";
    const method = isEdit ? "PUT" : "POST";
    
    const payload = {
        title,
        description,
        priority,
        due_date,
        tags,
        sub_tasks: subtasksToSend
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

if (addSubtaskBtn) {
    addSubtaskBtn.addEventListener("click", () => {
        // First, read current input values from DOM to preserve user typing
        const inputs = modalSubtasksContainer.querySelectorAll(".modal-subtask-item input");
        inputs.forEach((inp, idx) => {
            if (modalSubtasks[idx]) {
                modalSubtasks[idx].title = inp.value;
            }
        });
        
        // Add a new empty subtask
        modalSubtasks.push({
            id: 'sub_' + Math.random().toString(36).substr(2, 9),
            title: "",
            status: "pending"
        });
        
        renderModalSubtasks();
        
        // Focus on the newly added input
        const newInputs = modalSubtasksContainer.querySelectorAll(".modal-subtask-item input");
        if (newInputs.length > 0) {
            newInputs[newInputs.length - 1].focus();
        }
    });
}

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

// Tag filter select handler
tagFilter.addEventListener("change", (e) => {
    activeTag = e.target.value;
    loadTodos();
});

// Sort filter select handler
sortFilter.addEventListener("change", (e) => {
    const parts = e.target.value.split("_");
    if (parts.length >= 2) {
        activeSortOrder = parts.pop();
        activeSortBy = parts.join("_");
    }
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

// Search text highlighting helper with XSS protection and case sensitivity preservation
function highlightText(text, query) {
    if (!text) return "";
    if (!query) return escapeHTML(text);
    
    // Escape regex metacharacters in query safely
    const escapedQuery = query.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&');
    const regex = new RegExp(`(${escapedQuery})`, "gi");
    
    const parts = text.split(regex);
    
    return parts.map(part => {
        if (part.toLowerCase() === query.toLowerCase()) {
            return `<mark class="search-highlight">${escapeHTML(part)}</mark>`;
        }
        return escapeHTML(part);
    }).join("");
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
            const remainingText = formatRemainingTime(todo.due_date, todo.status);
            const displayRemaining = remainingText ? ` (${remainingText})` : "";
            dueDateSpan.innerText = dateObj.toLocaleDateString("id-ID", {
                day: "numeric",
                month: "long",
                year: "numeric",
                hour: "2-digit",
                minute: "2-digit"
            }) + displayRemaining;
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
    
    // Sub-tasks Card
    const detailSubtasksCard = document.getElementById("detail-subtasks-card");
    const detailSubtasksContainer = document.getElementById("detail-subtasks-container");
    if (detailSubtasksCard && detailSubtasksContainer) {
        if (todo.sub_tasks && todo.sub_tasks.length > 0) {
            detailSubtasksContainer.innerHTML = "";
            todo.sub_tasks.forEach((st) => {
                const isChecked = st.status === 'done';
                const stEl = document.createElement("div");
                stEl.className = `detail-subtask-item ${isChecked ? 'done' : ''}`;
                stEl.innerHTML = `
                    <div class="subtask-checkbox ${isChecked ? 'checked' : ''}"></div>
                    <span class="subtask-title">${escapeHTML(st.title)}</span>
                `;
                
                // Add click handler to toggle status
                stEl.addEventListener("click", async (e) => {
                    e.stopPropagation();
                    await toggleSubTaskStatus(todo.id, st.id);
                });
                
                detailSubtasksContainer.appendChild(stEl);
            });
            detailSubtasksCard.style.display = "block";
        } else {
            detailSubtasksCard.style.display = "none";
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

// ── SELECTION & BULK ACTIONS ──

// Toggle selection mode on/off
function toggleSelectionMode() {
    isSelectionMode = !isSelectionMode;
    selectedTodos.clear();
    
    if (isSelectionMode) {
        selectModeBtn.innerText = "Batal";
        selectModeBtn.classList.add("bold");
        bulkActionBar.classList.add("active");
        
        // Hide detail panel if it was open (Desktop UX)
        clearSelection();
    } else {
        selectModeBtn.innerText = "Pilih";
        selectModeBtn.classList.remove("bold");
        bulkActionBar.classList.remove("active");
    }
    
    updateBulkBarUI();
    renderTodoList(currentTodos); // Re-render to update click handlers and selection classes
}

// Toggle a single todo selection state
function toggleTodoSelection(id) {
    if (selectedTodos.has(id)) {
        selectedTodos.delete(id);
    } else {
        selectedTodos.add(id);
    }
    
    // Visually toggle selection state
    const todoEl = document.querySelector(`.todo-item[data-id="${id}"]`);
    if (todoEl) {
        todoEl.classList.toggle("selected-item");
    }
    
    updateBulkBarUI();
}

// Update the bulk bar count and state
function updateBulkBarUI() {
    const count = selectedTodos.size;
    bulkSelectCount.innerText = `${count} Terpilih`;
    
    // Enable/disable buttons based on selection size
    const disabled = count === 0;
    bulkCompleteBtn.disabled = disabled;
    bulkDeleteBtn.disabled = disabled;
    bulkCompleteBtn.style.opacity = disabled ? "0.5" : "1";
    bulkDeleteBtn.style.opacity = disabled ? "0.5" : "1";
}

// Handler for checkbox click (intercepts individual action if in selection mode)
function handleCheckboxClick(e, id, status) {
    e.stopPropagation();
    if (isSelectionMode) {
        toggleTodoSelection(id);
    } else {
        toggleComplete(id, status);
    }
}

// Trigger bulk complete
async function triggerBulkComplete() {
    if (selectedTodos.size === 0) return;
    
    try {
        const res = await request("/todos/bulk-complete", {
            method: "POST",
            body: JSON.stringify({ ids: Array.from(selectedTodos) })
        });
        
        if (res && res.success) {
            showToast(res.message || "Tugas berhasil diselesaikan secara massal!");
            toggleSelectionMode(); // Exit selection mode
            loadTodos();
        }
    } catch (err) {
        // handled
    }
}

// Trigger bulk delete
async function triggerBulkDelete() {
    if (selectedTodos.size === 0) return;
    if (!confirm(`Apakah Anda yakin ingin menghapus ${selectedTodos.size} tugas ini?`)) return;
    
    try {
        const res = await request("/todos/bulk-delete", {
            method: "POST",
            body: JSON.stringify({ ids: Array.from(selectedTodos) })
        });
        
        if (res && res.success) {
            showToast(res.message || "Tugas berhasil dihapus secara massal!");
            toggleSelectionMode(); // Exit selection mode
            loadTodos();
        }
    } catch (err) {
        // handled
    }
}

// Register Selection event listeners
selectModeBtn.addEventListener("click", toggleSelectionMode);
bulkCompleteBtn.addEventListener("click", triggerBulkComplete);
bulkDeleteBtn.addEventListener("click", triggerBulkDelete);

// ── THEME HANDLER ──
function initTheme() {
    const savedTheme = localStorage.getItem("todo_theme");
    const systemPrefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    const initialTheme = savedTheme || (systemPrefersDark ? "dark" : "light");
    
    applyTheme(initialTheme);
}

function applyTheme(theme) {
    if (theme === "dark") {
        document.documentElement.classList.add("dark-theme");
        document.body.classList.add("dark-theme");
        if (sunIcon) sunIcon.style.display = "block";
        if (moonIcon) moonIcon.style.display = "none";
    } else {
        document.documentElement.classList.remove("dark-theme");
        document.body.classList.remove("dark-theme");
        if (sunIcon) sunIcon.style.display = "none";
        if (moonIcon) moonIcon.style.display = "block";
    }
}

function toggleTheme() {
    const isDark = document.documentElement.classList.contains("dark-theme");
    const newTheme = isDark ? "light" : "dark";
    localStorage.setItem("todo_theme", newTheme);
    applyTheme(newTheme);
}

if (themeToggleBtn) {
    themeToggleBtn.addEventListener("click", toggleTheme);
}

// ── SCROLL OBSERVER ──
function initScrollObserver() {
    if (!scrollSentinel) return;
    
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting && hasMoreTodos && !isLoadingMore) {
                loadMoreTodos();
            }
        });
    }, {
        root: null, // browser viewport
        rootMargin: "0px 0px 100px 0px", // pre-fetch
        threshold: 0.1
    });
    
    observer.observe(scrollSentinel);
}

// ── INIT ──
initTheme();
initScrollObserver();
checkAuth();
