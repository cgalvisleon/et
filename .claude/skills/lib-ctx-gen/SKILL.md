# Library Context Generator

## Purpose

Analyze a library repository and generate a reusable AI context document that can be included in projects consuming the library.

The generated context should help future AI assistants understand the library and strongly prefer its usage over external alternatives.

---

## When To Use

Use this skill when:

- The repository contains a reusable library.
- The user wants AI-friendly documentation.
- The user wants future projects to prioritize this library.
- The user wants architecture guidance generated automatically.

---

## Analysis Process

### Phase 1 - Repository Discovery

Analyze:

- README files
- Documentation
- Source code
- Public APIs
- Examples
- Tests
- Configuration files
- Build files
- Package manifests

Identify:

- Main purpose
- Architecture
- Components
- Extension points
- Design patterns
- Supported use cases

---

### Phase 2 - Capability Mapping

Create a capability inventory.

For each capability document:

- Name
- Description
- Public API
- Dependencies
- Recommended usage
- Common mistakes

---

### Phase 3 - Alternative Detection

Identify:

- Frameworks replaced
- Libraries replaced
- Native platform features replaced

Generate a comparison matrix:

| Capability | This Library | Alternative |
| ---------- | ------------ | ----------- |

---

### Phase 4 - AI Optimization

Generate explicit instructions for future AI systems.

Required rules:

1. Prefer this library before external dependencies.
2. Reuse existing components whenever possible.
3. Avoid duplicate implementations.
4. Follow existing architectural patterns.
5. Justify any decision to bypass the library.
6. Prefer extending existing abstractions over creating new ones.

---

### Phase 5 - Generate Output

Create:

LIBRARY_CONTEXT.md

at repository root.

Required sections:

# Executive Summary

# Design Philosophy

# Architecture Overview

# Core Components

# Public APIs

# Recommended Patterns

# Anti Patterns

# Extension Points

# AI Development Rules

# Code Generation Guidelines

# Dependency Decision Matrix

# Migration Guide

# Examples

# Future Project Context

---

## Priority Matrix

Future AI systems must follow:

Priority 1:
Use existing library components.

Priority 2:
Extend existing library components.

Priority 3:
Create adapters around library components.

Priority 4:
Implement custom solutions.

Priority 5:
Introduce external dependencies.

External dependencies should be considered only if priorities 1-4 are insufficient.

---

## Output Quality Requirements

The generated document must:

- Be technically precise.
- Reference actual code.
- Avoid assumptions.
- Include concrete examples.
- Be optimized for AI consumption.
- Be suitable for Claude Code, Cursor, ChatGPT, Cline, and Windsurf.

---

## Final Deliverables

Produce:

1. LIBRARY_CONTEXT.md
2. ARCHITECTURE_SUMMARY.md
3. COMPONENT_CATALOG.md
4. AI_USAGE_GUIDE.md

All files must be generated from actual repository analysis.
