# Documentation Index

This directory contains comprehensive documentation for the Wavlake API. Start here to understand how everything works.

## 📋 **Quick Start**

1. **[CURRENT_ARCHITECTURE.md](CURRENT_ARCHITECTURE.md)** - How the API works today (start here)
2. **[README.md](README.md)** - Basic setup and development guide  
3. **[CLAUDE.md](CLAUDE.md)** - AI assistant context and development commands

## 🏗️ **Architecture Documentation**

### Current Production System
- **[CURRENT_ARCHITECTURE.md](CURRENT_ARCHITECTURE.md)** - Complete current architecture (GCS-based)

### API Features
- **[README_NOSTR_TRACKS.md](README_NOSTR_TRACKS.md)** - Nostr track upload API documentation
- **[LEGACY_API_TYPES.md](LEGACY_API_TYPES.md)** - Legacy PostgreSQL API types and interfaces

## 📁 **Document Organization**

### **Current State** (How things work today)
```
CURRENT_ARCHITECTURE.md          ← Complete current architecture document (GCS)
```

### **Development & Usage**
```
README.md                        ← Basic setup and usage
├── README_NOSTR_TRACKS.md       ← Track upload API examples
├── LEGACY_API_TYPES.md          ← TypeScript type definitions
└── CLAUDE.md                    ← Development assistant configuration
```

## 🎯 **What to Read Based on Your Goal**

### **I want to understand how the API works today**
→ Start with **[CURRENT_ARCHITECTURE.md](CURRENT_ARCHITECTURE.md)**

### **I want to set up the development environment**
→ Start with **[README.md](README.md)**

### **I want to use the track upload API**
→ Read **[README_NOSTR_TRACKS.md](README_NOSTR_TRACKS.md)**

### **I want to integrate with legacy data**
→ Read **[LEGACY_API_TYPES.md](LEGACY_API_TYPES.md)**

### **I want to work with Claude Code**
→ Read **[CLAUDE.md](CLAUDE.md)**

## 📊 **System Overview**

The Wavlake API provides:

- **Track Uploads**: NIP-98 authenticated uploads to Google Cloud Storage
- **Audio Processing**: FFmpeg-based compression with multiple format support
- **Legacy Data Access**: Read-only PostgreSQL endpoints for catalog API data
- **Dual Authentication**: Firebase JWT + Nostr NIP-98 signature support
- **Content Moderation**: Pubkey-based track ownership and removal capabilities

### Architecture Highlights
- **Storage**: Google Cloud Storage with `tracks/original/` and `tracks/compressed/` structure
- **Processing**: Cloud Functions trigger → API webhook → FFmpeg compression
- **Database**: Firestore (primary) + PostgreSQL (legacy read-only)
- **Deployment**: Cloud Run with VPC connector for secure database access

## ✅ **Documentation Quality Standards**

All documentation in this repository follows these standards:

- **Current**: Reflects production reality
- **Complete**: Covers setup, usage, and troubleshooting  
- **Clear**: No ambiguity about current vs future state
- **Organized**: Logical structure with clear navigation
- **Maintained**: Updated with code changes

## 🔄 **Document Lifecycle**

- **Current State Docs**: Updated immediately when code changes
- **API Documentation**: Updated when endpoints change
- **Development Docs**: Updated when development practices change

---

**Last Updated**: July 2025  
**Total Documents**: 5 core documents  
**Status**: All documentation reflects current GCS-based production system