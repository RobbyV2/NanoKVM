const id = {
  translation: {
    head: {
      desktop: 'Desktop jarak jauh',
      login: 'Masuk',
      changePassword: 'Ubah Sandi',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Masuk',
      placeholderUsername: 'Silahkan masukkan username',
      placeholderPassword: 'Silahkan masukkan password',
      placeholderCurrentPassword: 'Kata sandi saat ini',
      placeholderPassword2: 'Silahkan masukkan password again',
      noEmptyUsername: 'nama user tidak boleh kosong',
      noEmptyPassword: 'sandi  tidak boleh kosong',
      passwordLength: 'Kata sandi harus antara 8 dan 72 karakter',
      noAccount:
        'Gagal mendapatkan informasi user, silahkan segarkan halaman atau atur ulang sandi',
      invalidUser: 'invalid username or password',
      locked: 'Terlalu banyak login, silakan coba lagi nanti',
      globalLocked: 'Sistem dalam perlindungan, silakan coba lagi nanti',
      error: 'terjadi kesalahan tak terduga',
      invalidCurrentPassword: 'Kata sandi saat ini salah',
      changePassword: 'Ganti Sandi',
      changePasswordDesc: 'Untuk keamanan perangkat Anda, silakan ubah kata sandi masuk web.',
      differentPassword: 'sandi tidak sesuai',
      illegalUsername: 'ada karakter ilegal pada nama user',
      illegalPassword: 'ada karakter ilegal pada sandi',
      forgetPassword: 'Lupa Sandi',
      ok: 'Ok',
      cancel: 'Batalkan',
      loginButtonText: 'Masuk',
      tips: {
        reset1:
          'To reset the passwords, pressing and holding the BOOT button on the NanoKVM for 10 seconds.',
        reset2: 'Untuk langkah-langkah rinci, lihat dokumen ini:',
        reset3: 'Akun web default:',
        reset4: 'Akun SSH default:',
        change1: 'Perhatikan bahwa tindakan ini akan mengubah kata sandi berikut:',
        change2: 'Kata sandi login web',
        change3: 'Kata sandi root sistem (kata sandi login SSH)',
        change4: 'Untuk mengatur ulang kata sandi, tekan dan tahan tombol BOOT pada NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Konfigurasi Wi-Fi untuk NanoKVM',
      success: 'Please check the network status of NanoKVM and visit the new IP address.',
      failed: 'Operasi gagal, silakan coba lagi.',
      invalidMode:
        'Mode saat ini tidak mendukung pengaturan jaringan. Silakan buka perangkat Anda dan aktifkan mode konfigurasi Wi-Fi.',
      confirmBtn: 'Ok',
      finishBtn: 'Selesai',
      ap: {
        authTitle: 'Otentikasi Diperlukan',
        authDescription: 'Silakan masukkan kata sandi AP untuk melanjutkan',
        authFailed: 'Kata sandi AP tidak valid',
        passPlaceholder: 'AP kata sandi',
        verifyBtn: 'Verifikasi'
      }
    },
    screen: {
      scale: 'Skala',
      title: 'Layar',
      video: 'Mode Video',
      videoDirectTips: 'Aktifkan HTTPS di "Pengaturan > Perangkat" untuk menggunakan mode ini',
      resolution: 'Resolusi',
      auto: 'Otomatis',
      autoTips:
        'Tearing layar atau offset tetikus dapat terjadi pada resolusi tertentu. Pertimbangkan untuk menyesuaikan resolusi host jarak jauh atau menonaktifkan mode otomatis.',
      fps: 'FPS',
      customizeFps: 'Sesuaikan',
      quality: 'Kualitas',
      qualityLossless: 'Tanpa Kehilangan',
      qualityHigh: 'Tinggi',
      qualityMedium: 'Sedang',
      qualityLow: 'Rendah',
      frameDetect: 'Deteksi bingkai',
      frameDetectTip:
        'Hitung selisih antar frame. Hentikan transmisi aliran video saat tidak ada perubahan yang terdeteksi di layar host jarak jauh.',
      resetHdmi: 'Atur ulang HDMI',
      mixedH264: {
        title: 'Konflik aliran H.264',
        description:
          'H.264 Direct dan H.264 WebRTC sedang digunakan secara bersamaan. Hal ini dapat menyebabkan layar robek atau video rusak. Harap gunakan hanya satu mode H.264.'
      },
      webrtcConnectionFailed: {
        title: 'Koneksi WebRTC gagal',
        description: 'Periksa koneksi jaringan atau ganti mode video.'
      },
      captureStatus: {
        hdmiError: 'Kesalahan layar HDMI',
        unsupportedResolution: 'Resolusi saat ini tidak didukung',
        retrieving: 'Mengambil layar...',
        changingResolution: 'Mengganti resolusi...',
        updateFailed: 'Layar tidak dapat diperbarui saat ini',
        videoError: 'Kesalahan tampilan video',
        noHdmi: 'Sinyal HDMI tidak terdeteksi',
        unavailable: 'Layar tidak dapat ditampilkan saat ini'
      }
    },
    keyboard: {
      title: 'Keyboard',
      paste: 'Tempel',
      tips: 'Hanya huruf dan simbol keyboard standar yang didukung',
      placeholder: 'Silahkan isi',
      submit: 'Kirimkan',
      virtual: 'Keyboard',
      readClipboard: 'Membaca dari Papan Klip',
      clipboardPermissionDenied:
        'Izin papan klip ditolak. Harap izinkan akses clipboard di browser Anda.',
      clipboardReadError: 'Gagal membaca papan klip',
      dropdownEnglish: 'Bahasa Inggris',
      dropdownGerman: 'Jerman',
      dropdownFrench: 'Perancis',
      dropdownRussian: 'Rusia',
      shortcut: {
        title: 'Pintasan',
        custom: 'Adat',
        capture: 'Klik di sini untuk mengambil pintasan',
        clear: 'Jelas',
        save: 'Simpan',
        captureTips:
          'Menangkap tombol tingkat sistem (seperti tombol Windows) memerlukan izin layar penuh.',
        enterFullScreen: 'Beralih ke mode layar penuh.'
      },
      leaderKey: {
        title: 'Tombol Leader',
        desc: 'Lewati batasan browser dan kirim pintasan sistem langsung ke host jarak jauh.',
        howToUse: 'Cara Menggunakan',
        simultaneous: {
          title: 'Mode Simultan',
          desc1: 'Tekan dan tahan tombol Leader, lalu tekan pintasan.',
          desc2: 'Intuitif, tetapi mungkin bertentangan dengan pintasan sistem.'
        },
        sequential: {
          title: 'Mode Berurutan',
          desc1:
            'Tekan tombol Leader → tekan pintasan secara berurutan → tekan tombol Leader lagi.',
          desc2: 'Memerlukan lebih banyak langkah, namun sepenuhnya menghindari konflik sistem.'
        },
        enable: 'Aktifkan tombol Leader',
        tip: 'Saat ditetapkan sebagai tombol Leader, tombol ini hanya berfungsi sebagai pemicu pintasan dan kehilangan perilaku defaultnya.',
        placeholder: 'Tekan tombol Leader',
        shiftRight: 'Shift kanan',
        ctrlRight: 'Ctrl kanan',
        metaRight: 'Win kanan',
        submit: 'Kirimkan',
        recorder: {
          rec: 'REKAM',
          activate: 'Aktifkan tombol',
          input: 'Silakan tekan pintasan...'
        }
      }
    },
    mouse: {
      title: 'Tikus',
      cursor: 'Gaya kursor',
      default: 'Kursor bawaan',
      pointer: 'Kursor penunjuk',
      cell: 'Kursor cell',
      text: 'Kursor teks',
      grab: 'Kursor ambil',
      hide: 'Sembunyikan kursor',
      mode: 'Mode tetikus',
      absolute: 'Mode absolut',
      relative: 'Mode relatif',
      direction: 'Arah roda gulir',
      scrollUp: 'Gulir ke atas',
      scrollDown: 'Gulir ke bawah',
      speed: 'Kecepatan roda gulir',
      fast: 'Cepat',
      slow: 'Lambat',
      requestPointer:
        'Menggunakan mode relatf. Silakan klik desktop untuk mendapatkan penunjuk tetikus.',
      resetHid: 'Setel ulang HID',
      hidOnly: {
        title: 'Mode hanya HID',
        desc: 'Jika mouse dan keyboard Anda berhenti merespons dan menyetel ulang HID tidak membantu, mungkin ada masalah kompatibilitas antara NanoKVM dan perangkat. Coba aktifkan mode HID-Only untuk kompatibilitas yang lebih baik.',
        tip1: 'Mengaktifkan mode HID-Hanya akan melepas U-disk virtual dan jaringan virtual',
        tip2: 'Dalam mode HID-Only, pemasangan gambar dinonaktifkan',
        tip3: 'NanoKVM akan otomatis reboot setelah berpindah mode',
        enable: 'Aktifkan mode HID-Hanya',
        disable: 'Nonaktifkan mode HID-Hanya'
      }
    },
    image: {
      title: 'Gambar',
      loading: 'Memuat...',
      empty: 'Tidak ada yang ditemukan',
      mountMode: 'Mode pemasangan',
      mountFailed: 'Pemasangan Gagal',
      mountDesc:
        'Di beberapa sistem, perlu mengeluarkan disk virtual pada host jarak jauh sebelum memasang gambar.',
      unmountFailed: 'Pelepasan gagal',
      unmountDesc:
        'Pada beberapa sistem, Anda perlu mengeluarkan secara manual dari host jarak jauh sebelum melepas gambar.',
      refresh: 'Segarkan daftar gambar',
      attention: 'Perhatian',
      deleteConfirm: 'Apakah Anda yakin ingin menghapus gambar ini?',
      okBtn: 'Ya',
      cancelBtn: 'Tidak',
      tips: {
        title: 'Cara mengunggah',
        usb1: 'Hubungkan NanoKVM ke komputer Anda melalui USB.',
        usb2: 'Pastikan disk virtual telah terpasang (Pengaturan - Disk Virtual).',
        usb3: 'Buka disk virtual di komputer Anda dan salin file gambar ke direktori root disk virtual.',
        scp1: 'Pastikan NanoKVM dan komputer Anda berada di jaringan lokal yang sama.',
        scp2: 'Buka terminal di komputer Anda dan gunakan perintah SCP untuk mengunggah file gambar ke direktori /data di NanoKVM.',
        scp3: 'Contoh: scp jalur-gambar-anda root@ip-nanokvm-anda:/data',
        tfCard: 'Kartu TF',
        tf1: 'Metode ini didukung di sistem linux',
        tf2: 'Dapatkan Kartu TF dari NanoKVM (untuk versi LENGKAP, bongkar casingnya terlebih dahulu).',
        tf3: 'Masukkan Kartu TF ke pembaca kartu dan hubungkan ke komputer Anda.',
        tf4: 'Salin berkas gambar ke direktori /data pada Kartu TF.',
        tf5: 'Masukkan Kartu TF ke dalam NanoKVM.'
      }
    },
    script: {
      title: 'Script',
      upload: 'Mengunggah',
      run: 'Jalankan',
      runBackground: 'Jalankan di belakang',
      runFailed: 'Gagal menjalankan',
      attention: 'Perhatian',
      delDesc: 'Apa kamu yakin menghapus data ini?',
      confirm: 'Ya',
      cancel: 'Tidak',
      delete: 'Hapus',
      close: 'Tutup'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal NanoKVM',
      serial: 'Terminal Port Serial',
      serialPort: 'Port Serial',
      serialPortPlaceholder: 'Silahkan masukkan port serial',
      baudrate: 'Baud rate',
      parity: 'Paritas',
      parityNone: 'Tidak ada',
      parityEven: 'Genap',
      parityOdd: 'Ganjil',
      flowControl: 'Kontrol aliran',
      flowControlNone: 'Tidak ada',
      flowControlSoft: 'Perangkat lunak',
      flowControlHard: 'Perangkat keras',
      dataBits: 'Bit data',
      stopBits: 'Hentikan bit',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Kirim perintah...',
      sent: 'Perintah terkirim',
      input: 'Silahkan masukkan MAC',
      ok: 'Ok'
    },
    download: {
      title: 'Pengunduh Gambar',
      input: 'Silakan masukkan gambar jarak jauh URL',
      ok: 'Ok',
      disabled: 'Partisi /data adalah RO, jadi kami tidak dapat mengunduh gambarnya',
      uploadbox: 'Letakkan file di sini atau klik untuk memilih',
      inputfile: 'Silakan masukkan File gambar',
      NoISO: 'Tidak ada ISO',
      sha256: 'SHA-256 (opsional)',
      sha256Placeholder: 'Masukkan checksum SHA-256 64 karakter',
      invalidSHA256: 'SHA-256 harus berupa string heksadesimal 64 karakter',
      failed: 'Unduhan gagal',
      success: 'Unduhan berhasil',
      checksumFailed: 'Unduhan gagal: verifikasi SHA-256 gagal',
      cancel: 'Batal',
      cancelFailed: 'Gagal membatalkan unduhan'
    },
    power: {
      title: 'Daya',
      showConfirm: 'Konfirmasi',
      showConfirmTip: 'Pengoperasian listrik memerlukan konfirmasi tambahan',
      reset: 'Mulai Ulang',
      power: 'Daya',
      powerShort: 'Data (tekan sebentar)',
      powerLong: 'Power (tekan lama)',
      resetConfirm: 'Lanjutkan operasi penyetelan ulang?',
      powerConfirm: 'Lanjutkan pengoperasian listrik?',
      okBtn: 'Ya',
      cancelBtn: 'Tidak'
    },
    devices: {
      title: 'Perangkat',
      stale: 'Status langsung perangkat tidak tersedia. Menyambung ulang.',
      empty:
        'Belum ada slot kamera atau mikrofon yang dikonfigurasi. Tambahkan di Pengaturan, Perangkat.',
      available: 'Tersedia',
      waiting: 'Host menunggu sebuah sumber',
      hostOpen: 'Host terbuka',
      hostIdle: 'Host menganggur',
      sending: 'Mengirim dari peramban ini',
      black: 'Video hitam',
      silence: 'Senyap digital',
      resuming: 'Menunggu untuk dilanjutkan',
      stop: 'Hentikan berbagi',
      disconnect: 'Putuskan',
      takeover: 'Ambil alih',
      refused: 'Sedang dipakai oleh {{owner}} dari {{source}}',
      connectedSources_one: '{{count}} sumber tersambung',
      connectedSources_other: '{{count}} sumber tersambung',
      connectedSources: '{{count}} sumber tersambung',
      connection: {
        connecting: 'Menyambungkan',
        connected: 'Langsung',
        disconnected: 'Menyambung ulang'
      },
      share: {
        camera: 'Bagikan kamera',
        microphone: 'Bagikan mikrofon',
        usbDevice: 'Bagikan USB'
      },
      permission: {
        denied: 'Diblokir di pengaturan situs peramban Anda',
        prompt: 'Peramban akan meminta izin akses'
      },
      mic: {
        mute: 'Bisukan',
        unmute: 'Bunyikan'
      },
      revoked: {
        released: 'Berbagi dihentikan',
        lease_expired: 'Masa sewa habis sebelum peramban ini kembali',
        admin_disconnect: 'Seorang administrator memutuskan semua sumber',
        slot_removed: 'Slot dihapus',
        slot_changed: 'Slot dikonfigurasi ulang',
        taken_over: 'Seorang administrator mengambil alih slot ini'
      },
      usb: {
        surrendered: 'Passthrough USB sedang memegang papan ketik dan tetikus',
        surrenderedDesc:
          'Host jarak jauh melihat perangkat yang diimpor, bukan papan ketik, tetikus, dan media virtual NanoKVM. Semuanya kembali begitu sesi berhenti.',
        unsupported: 'WebUSB memerlukan peramban berbasis Chromium',
        insecure:
          'Halaman ini tidak disajikan lewat HTTPS, sehingga peramban menyembunyikan WebUSB. Aktifkan HTTPS di Pengaturan, Jaringan.',
        session: 'Meneruskan {{device}} ({{mode}})',
        idle: 'Tidak ada sesi passthrough',
        mode: {
          hybrid: 'hibrida',
          exact: 'persis'
        }
      }
    },
    settings: {
      title: 'Pengaturan',
      display: {
        title: 'Tampilan',
        loading: 'Memuat...',
        active: 'EDID aktif',
        activeUnknown:
          'NanoKVM belum menulis EDID sejak dinyalakan, jadi identitas monitor yang dilihat host tidak diketahui.',
        appliedAt: 'Diterapkan {{time}}',
        download: 'Unduh',
        downloadBackup: 'Unduh yang sebelumnya',
        preset: 'Preset monitor',
        presetPlaceholder: 'Pilih monitor',
        upload: 'Unggah',
        selected: 'EDID terpilih',
        errors: 'Kesalahan',
        warnings: 'Peringatan',
        info: 'Informasi',
        unknownMonitor: 'Monitor tidak dikenal',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Ada audio',
        audioNo: 'Tanpa audio',
        extensionBlocks: 'Blok ekstensi: {{blocks}}',
        apply: 'Terapkan',
        applyTitle: 'Terapkan EDID ini?',
        before: 'Saat ini',
        after: 'Baru',
        hdmiNotice:
          'Pengambilan video berhenti selama EDID ditulis dan berjalan lagi dengan sendirinya setelah selesai.',
        powerCycleNotice:
          'Perangkat ini harus dicabut dari listrik lalu dicolokkan kembali agar EDID baru berlaku.',
        powerCycleUnverified:
          'Penulisan tidak terverifikasi, jadi cip video menyimpan apa pun yang ada di dalamnya sekarang sampai perangkat ini dicabut dari daya secara fisik lalu dipasang kembali.',
        applied: 'EDID diterapkan dan terverifikasi.',
        applyFailed: 'Gagal menerapkan EDID.',
        busy: 'Chip video sedang sibuk. Coba lagi.',
        unsupported: 'Perangkat ini tidak mendukung penggantian EDID.',
        toolMissing: 'Alat EDID tidak ada di firmware ini.',
        noAudio: 'EDID ini tidak mengumumkan audio, jadi host mungkin berhenti mengirim suara.',
        oldVersion: 'EDID ini memakai versi lebih lama dari 1.4.',
        interlaced: 'Timing pilihan bersifat interlaced.',
        tooLarge:
          'Timing pilihan lebih besar dari 1920x1080 pada 60 Hz, melebihi kemampuan tangkap NanoKVM.',
        recovery: 'Pemulihan',
        recoveryNeeded:
          'Penulisan terakhir tidak terverifikasi, sehingga area EDID pada chip video berada dalam keadaan yang tidak diketahui. Pulihkan EDID pabrik agar keadaannya diketahui kembali.',
        recoveryDesc:
          'Tulis kembali EDID yang diketahui ke chip video ketika EDID yang diterapkan membuat host tidak menampilkan gambar.',
        restoreFactory: 'Pulihkan EDID pabrik',
        restoreBackup: 'Pulihkan EDID sebelumnya',
        restoreTitle: 'Pulihkan EDID ini?',
        restoreFactoryTarget: 'EDID pabrik yang disertakan bersama NanoKVM.',
        restoreBackupTarget: 'Cadangan terbaru, diterapkan {{time}}.',
        restoreNotice:
          'Pemulihan ditulis dengan cara yang sama seperti penerapan, dengan konsekuensi yang sama.',
        restored: 'EDID dipulihkan dan diverifikasi.',
        restoreFailed: 'Gagal memulihkan EDID.',
        mismatchTitle: 'Yang ditulis dan yang dibaca kembali',
        mismatchWritten: 'Ditulis',
        mismatchRead: 'Dibaca kembali',
        restoreOkBtn: 'Pulihkan',
        hardware: 'Perangkat keras terdeteksi: {{hardware}}',
        hardwareUnknown: 'Tidak diketahui',
        confirmWord: 'TERAPKAN',
        confirmPrompt: 'Ketik {{word}} untuk mengaktifkan tombol terapkan.',
        okBtn: 'Terapkan',
        cancelBtn: 'Batal'
      },
      presentation: {
        title: 'Presentasi USB',
        loading: 'Memuat...',
        current: 'Presentasi USB saat ini',
        noProfile: 'Tidak ada profil yang diterapkan',
        linked: 'Fungsi yang tertaut',
        hostState: 'USB host',
        hostUnbound: 'Pengendali tidak terikat',
        hdmiState: 'Masukan HDMI',
        hdmiSignal: 'Ada sinyal',
        hdmiUnreported: 'Belum ada laporan penangkapan',
        endpoints: 'Endpoint',
        fifos: 'Slot FIFO',
        pending: 'Perubahan tertunda',
        pendingEdits: 'Suntingan identitas yang belum disimpan',
        pendingProfile: '{{profile}} dipilih tetapi belum diterapkan',
        pendingNone: 'Tidak ada',
        lastApply: 'Penerapan terakhir',
        applyFailed: 'Gagal pada {{profile}} pukul {{time}}',
        applyClean: 'Tidak ada kegagalan tercatat',
        lastKnownGood: 'Kondisi baik terakhir',
        rollbackTarget: 'Target pengembalian',
        rollbackNone: 'Tidak ada',
        powerCyclePending:
          'Pengendali diambil dari host. Matikan lalu nyalakan kembali komputer yang terhubung untuk mendapatkan perangkatnya lagi.',
        rollback: 'Kembalikan',
        rollbackTitle: 'Kembalikan ke {{profile}}?',
        rollbackDesc: 'Gadget dienumerasi ulang; fungsi USB terputus sesaat.',
        profile: 'Profil USB',
        builtIn: 'bawaan',
        descriptors: 'deskriptor',
        imported: 'diimpor',
        clone: 'Klon',
        cloneTitle: 'Klon profil ini',
        cloneToEdit:
          'Profil bawaan tetap hanya-baca. Klon profil ini untuk menyunting identitasnya.',
        profileName: 'Nama profil',
        profileNameHint: 'Huruf kecil, angka, titik, garis bawah, dan tanda hubung.',
        import: 'Impor paket',
        export: 'Ekspor paket',
        delete: 'Hapus',
        deleteTitle: 'Hapus profil ini?',
        deleteDesc: 'Hapus {{profile}} dari NanoKVM.',
        identity: 'Identitas USB',
        preset: 'Identitas preset',
        presetPlaceholder: 'Salin identitas dari perangkat yang dikenal',
        presetHint:
          'Preset mengisi Vendor ID, Product ID, dan kedua kolom nama. Preset tidak membawa deskriptor.',
        presetSource: 'Identitas seperti yang tercatat di {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Vendor ID ini milik produsen lain',
        productId: 'Product ID',
        bcdUSB: 'Versi USB',
        bcdDevice: 'Versi perangkat',
        manufacturer: 'Produsen',
        product: 'Produk',
        serial: 'Nomor seri',
        configuration: 'Untai konfigurasi',
        hidLayout: 'Perangkat HID',
        hidRoleKeyboard: 'Papan ketik',
        hidRoleRelative: 'Tetikus (relatif)',
        hidRoleAbsolute: 'Penunjuk (absolut)',
        hidOff: 'Tidak ada',
        hidInterface: 'Antarmuka {{index}}',
        hidBootKeyboardShared:
          'Papan ketik berbagi satu antarmuka, jadi tidak lagi menyediakan laporan protokol boot. Sebagian BIOS dan UEFI tidak akan melihatnya.',
        functions: 'Fungsi',
        descriptorAssets: 'Berkas deskriptor tersimpan: {{count}}',
        endpointUse:
          'IN {{inUse}} terpakai, {{inFree}} bebas; OUT {{outUse}} terpakai, {{outFree}} bebas',
        apply: 'Terapkan',
        applyTitle: 'Terapkan profil USB ini?',
        applyDesc: 'NanoKVM akan menampilkan {{profile}} ke komputer yang tersambung.',
        reconnect:
          'Papan ketik, tetikus, dan fungsi USB lainnya terputus sesaat saat gadget diikat ulang.',
        applyLinks: 'Menautkan: {{functions}}',
        applyRemoves: 'Menghapus: {{functions}}',
        applyNoHid:
          'Tidak ada fungsi HID yang tersisa setelah penerapan ini. Papan ketik dan tetikus akan berhenti bekerja.',
        applyRollback: 'Penerapan yang gagal akan kembali ke {{profile}}.',
        recoveryPowerCycle:
          'Tidak ada HID yang bertahan dari penerapan ini, jadi host yang berhenti merespons hanya bisa dipulihkan dengan mematikan dan menyalakan dayanya.',
        recoveryReboot:
          'Sebuah antarmuka hilang dari perangkat komposit; host mungkin perlu di-boot ulang agar sisanya terikat kembali.',
        recoveryHdmiReset:
          'Sebuah fungsi video dibangun ulang, sehingga alur penangkapan di belakangnya ikut direset.',
        recoveryReconnect: 'Host mengenumerasi ulang perangkat; fungsi USB terputus sesaat.',
        cancel: 'Batal',
        noFunctions: 'Tidak ada fungsi yang tertaut',
        loadFailed: 'Gagal memuat profil presentasi',
        operationFailed: 'Operasi presentasi gagal'
      },
      passthrough: {
        title: 'Passthrough USB',
        loading: 'Memuat...',
        mode: 'Mode',
        hybrid: 'Hibrida',
        exact: 'Persis',
        hybridDesc:
          'Mempertahankan papan ketik boot dan tetikus relatif, untuk perangkat yang kompatibel.',
        exactDesc: 'Mengganti setiap fungsi USB NanoKVM dengan perangkat yang diteruskan.',
        hybridWarning: 'Mode hibrida menjaga papan ketik dan tetikus relatif tetap tersedia',
        hybridWarningDesc:
          'Penyimpanan, jaringan USB, dan penunjuk absolut terputus selama fungsi yang diteruskan aktif.',
        hidWarning: 'Memulai passthrough menyerahkan keyboard, mouse, dan media virtual',
        hidWarningDesc:
          'NanoKVM hanya punya satu pengontrol perangkat USB dan proxy membutuhkannya sepenuhnya, sehingga selama sesi berjalan host jarak jauh melihat perangkat yang diteruskan alih-alih keyboard, mouse, dan media virtual NanoKVM. Semuanya kembali dengan sendirinya begitu sesi dihentikan. Antarmuka web ini tidak terpengaruh, jadi Anda selalu bisa menghentikan sesi dari halaman ini.',
        hidWarningSafeDesc:
          'NanoKVM hanya punya satu pengontrol perangkat USB dan proxy membutuhkannya sepenuhnya, sehingga selama sesi berjalan host jarak jauh melihat perangkat yang diteruskan alih-alih keyboard, mouse, dan media virtual NanoKVM. Semuanya kembali saat sesi dihentikan.',
        isoLabel: 'Izinkan transfer isokron',
        isoHint:
          'Memasukkan webcam, mikrofon, dan perangkat aliran lain. Belum ada yang mengukur kemampuan perangkat keras ini.',
        isoWarning:
          'Aliran isokron belum terbukti di sini dan dapat menahan papan ketik serta tetikus sampai Anda menghentikan sesi',
        info: {
          title: 'Info',
          hybrid:
            'Mode hibrida menjaga papan ketik dan tetikus relatif tetap tersedia. Penyimpanan, jaringan USB, dan penunjuk absolut terputus selama perangkat yang diteruskan aktif.',
          exact:
            'Mode persis mengganti setiap fungsi USB NanoKVM dengan perangkat yang diteruskan. Papan ketik, tetikus, dan media virtual kembali dengan sendirinya saat sesi dihentikan.',
          udc: 'NanoKVM hanya punya satu pengontrol perangkat USB dan proxy membutuhkannya sepenuhnya; itulah sebabnya fungsi di atas menghilang selama sesi berjalan.',
          web: 'Antarmuka web ini tidak terpengaruh, jadi Anda selalu bisa menghentikan sesi dari halaman ini.',
          network:
            'Mulai passthrough lewat Ethernet atau Wi-Fi. Memulainya dari jaringan USB NanoKVM ditolak, karena koneksi itu akan hilang.',
          iso: 'Webcam, mikrofon, dan perangkat isokron lain ditolak selama Anda tidak mengizinkan transfer isokron. Jalur itu bekerja tetapi belum pernah diukur pada perangkat keras ini, jadi anggap lajunya tidak diketahui.',
          camera:
            'Kamera dan mikrofon peramban di bagian Perangkat tetap menjadi cara yang terbukti untuk memberikannya kepada host.'
        },
        session: 'Sesi',
        activeDesc: 'Sebuah perangkat telah diimpor dan proxy sedang memegang pengontrol USB.',
        inactiveDesc:
          'Tidak ada sesi yang berjalan. Keyboard, mouse, dan media virtual bekerja normal.',
        device: 'Perangkat',
        busId: 'ID bus',
        speed: 'Kecepatan',
        exporter: 'Pengekspor',
        local: 'Diimpor sebagai',
        localValue: 'Bus {{bus}}, alamat {{address}}',
        udc: 'Pengontrol USB',
        pid: 'PID proxy',
        startedAt: 'Dimulai',
        isoDevice:
          'Perangkat ini mengalirkan data lewat endpoint isokron, yang belum pernah diukur pada perangkat keras ini',
        exporterLabel: 'Alamat pengekspor',
        exporterHint:
          'Host dan porta yang dihubungi NanoKVM. Melalui terowongan di bawah, itu adalah {{exporter}}.',
        busIdLabel: 'ID bus di komputer Anda',
        busIdHint:
          'Busid yang ditampilkan usbip list -l untuk perangkat tersebut, misalnya {{example}}.',
        start: 'Mulai passthrough',
        stop: 'Hentikan passthrough',
        startTitle: 'Mulai passthrough USB?',
        startDevice: 'NanoKVM akan mengimpor {{busId}} dari {{exporter}}.',
        startHid:
          'Keyboard USB, mouse, dan media virtual berhenti bekerja selama sesi berjalan, dan kembali dengan sendirinya saat Anda menghentikannya.',
        startIso:
          'Webcam dan perangkat isokron lain memerlukan sakelar isokron dinyalakan sebelum Anda mulai.',
        startWeb:
          'Antarmuka web ini tetap berjalan, jadi Anda dapat menghentikan sesi dari halaman ini kapan saja.',
        startNetwork:
          'Gunakan halaman ini lewat Ethernet atau Wi-Fi. Memulai dari jaringan USB NanoKVM ditolak karena koneksi itu akan hilang.',
        okBtn: 'Mulai',
        cancelBtn: 'Batal',
        instructions: 'Di komputer Anda sendiri',
        instructionsDesc:
          'Memang tidak ada agen klien yang perlu dipasang. Jalankan perintah usbip standar berikut di komputer tempat perangkat terpasang.',
        copyFailed: 'Gagal menyalin. Salin perintah secara manual.',
        copyInsecure:
          'Halaman ini tidak disajikan lewat HTTPS, sehingga peramban memblokir penyalinan. Salin perintah secara manual, atau aktifkan HTTPS di Pengaturan, Jaringan.',
        directNote:
          'Tanpa terowongan, usbipd harus dapat dijangkau di jaringan Anda dan alamat pengekspor di atas harus menunjuk ke sana. usbip membawa data perangkat tanpa enkripsi, jadi terowongan lebih disarankan.',
        steps: {
          modprobe: {
            title: 'Muat driver sisi pengekspor',
            desc: 'usbip-host memungkinkan kernel menyerahkan perangkat lokal. Modul ini tidak dimuat secara bawaan.'
          },
          list: {
            title: 'Temukan perangkatnya',
            desc: 'Menampilkan setiap perangkat lokal beserta busid dan pasangan vendor:produk. Catat busid perangkat yang Anda inginkan.'
          },
          bind: {
            title: 'Ikat ke usbip',
            desc: 'Mengambil perangkat dari driver normalnya, sehingga perangkat berhenti bekerja di komputer ini sampai Anda melepas ikatannya.'
          },
          serve: {
            title: 'Sajikan perangkatnya',
            desc: 'usbipd tetap berjalan di latar depan dan menunggu NanoKVM mengimpor perangkat.',
            notice:
              'usbipd standar tidak punya opsi alamat dengar dan mendengarkan di semua antarmuka. Tutup porta {{port}} di firewall Anda dan biarkan hanya terowongan di bawah yang menjangkaunya.'
          },
          tunnel: {
            title: 'Arahkan ke NanoKVM',
            desc: 'Terowongan balik SSH: porta {{port}} pada loopback NanoKVM sendiri menjadi usbipd di komputer ini. Biarkan berjalan selama sesi berlangsung.'
          },
          exporter: {
            title: 'Gunakan ini sebagai pengekspor',
            desc: 'Masukkan ini ke kolom pengekspor di atas, isi ID bus, lalu mulai sesinya.'
          },
          unbind: {
            title: 'Kembalikan perangkatnya',
            desc: 'Setelah sesi dihentikan, perintah ini mengembalikan perangkat ke driver normalnya di komputer ini.'
          }
        }
      },
      mcp: {
        title: 'Layanan MCP',
        service: 'Kontrol jarak jauh MCP',
        serviceDesc:
          'Izinkan klien MCP tepercaya mengontrol keyboard dan mouse serta mengambil tangkapan layar',
        securityWarning:
          'Siapa pun yang memiliki kunci API ini dapat mengontrol host jarak jauh dan melihat layarnya. Gunakan HTTPS dan aktifkan hanya pada jaringan tepercaya.',
        endpoint: 'Endpoint',
        apiKey: 'Kunci API',
        regenerateConfirmTitle: 'Buat ulang kunci API MCP?',
        regenerateConfirmDesc: 'Kunci saat ini akan langsung berhenti berfungsi.',
        enableConfirmTitle: 'Aktifkan kontrol MCP eksternal?',
        enableConfirmDesc:
          'Mengaktifkan MCP akan menghentikan PicoClaw dan menutup semua sesi PicoClaw yang aktif.',
        failed: 'Operasi MCP gagal',
        copyFailed: 'Gagal menyalin. Salin secara manual.',
        copyInsecure:
          'Halaman ini tidak disajikan lewat HTTPS, sehingga peramban memblokir penyalinan. Salin secara manual, atau aktifkan HTTPS di Pengaturan, Jaringan.',
        okBtn: 'Konfirmasi',
        cancelBtn: 'Batal'
      },
      about: {
        title: 'Tentang NanoKVM',
        information: 'Informasi',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Versi Aplikasi',
        applicationTip: 'Versi aplikasi web NanoKVM',
        image: 'Version Gambar',
        imageTip: 'Versi image sistem NanoKVM',
        deviceKey: 'Kunci Perangkat',
        community: 'Komunitas',
        hostname: 'Nama Host',
        hostnameUpdated: 'Nama host diperbarui. Nyalakan ulang untuk menerapkan.',
        ipType: {
          Wired: 'Berkabel',
          Wireless: 'Nirkabel',
          Other: 'Lainnya'
        }
      },
      appearance: {
        title: 'Tampilan',
        display: 'Layar',
        language: 'Bahasa',
        languageDesc: 'Pilih bahasa untuk antarmuka',
        webTitle: 'Judul Web',
        webTitleDesc: 'Menyesuaikan judul halaman web',
        favicon: 'Favicon',
        faviconDesc: 'Menyesuaikan ikon tab peramban',
        faviconPlaceholder: 'URL gambar',
        faviconUpload: 'Unggah',
        faviconReset: 'Setel ulang',
        faviconCustom: 'Ikon khusus',
        faviconBoot: 'Ikon dari /boot/logo.ico',
        faviconDefault: 'Ikon bawaan',
        faviconOverridesBoot: 'Menimpa /boot/logo.ico',
        faviconErrUrl: 'Masukkan alamat gambar http:// atau https://',
        faviconErrFetch: 'Perangkat tidak dapat mengunduh gambar tersebut',
        faviconErrLarge: 'Gambar terlalu besar. Batasnya 256 KB',
        faviconErrType: 'Gambar tidak didukung. Gunakan .ico, .png, .svg, .gif atau .jpg',
        faviconErrSave: 'Gagal menyimpan ikon',
        menuBar: {
          title: 'Bilah Menu',
          mode: 'Mode Tampilan',
          modeDesc: 'Menampilkan bilah menu di layar',
          modeOff: 'Mati',
          modeAuto: 'Sembunyikan otomatis',
          modeAlways: 'Selalu terlihat',
          keyboardLedStatus: 'Indikator kunci keyboard',
          keyboardLedStatusDesc:
            'Tampilkan status Num Lock, Caps Lock, dan Scroll Lock komputer jarak jauh',
          icons: 'Ikon Submenu',
          iconsDesc: 'Menampilkan ikon submenu di bilah menu'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Status kunci keyboard jarak jauh',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'Aktif',
        off: 'Nonaktif',
        unknown: 'Tidak diketahui'
      },
      device: {
        title: 'Perangkat',
        oled: {
          title: 'OLED',
          description: 'OLED screen automatically sleep',
          0: 'Tidak pernah',
          15: '15 sec',
          30: '30 sec',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 jam'
        },
        ssh: {
          description: 'Aktifkan akses jarak jauh SSH',
          tip: 'Tetapkan kata sandi yang kuat sebelum mengaktifkan (Akun - Ubah Kata Sandi)'
        },
        advanced: 'Pengaturan Lanjutan',
        swap: {
          title: 'Tukar',
          disable: 'Nonaktifkan',
          description: 'Atur ukuran file swap',
          tip: 'Mengaktifkan fitur ini dapat mempersingkat masa pakai kartu SD Anda!'
        },
        mouseJiggler: {
          title: 'Tikus Jiggler',
          description: 'Mencegah host jarak jauh tertidur',
          disable: 'Nonaktifkan',
          absolute: 'Mode Absolut',
          relative: 'Mode Relatif'
        },
        mdns: {
          description: 'Aktifkan layanan penemuan mDNS',
          tip: 'Mematikan jika tidak diperlukan'
        },
        hdmi: {
          description: 'Aktifkan keluaran HDMI/monitor',
          idleTimeoutTitle: 'Batas waktu tangkapan tidak aktif',
          idleTimeoutDescription: 'Hentikan tangkapan HDMI setelah tidak ada penonton aktif selama',
          minutes: 'mnt'
        },
        autostart: {
          title: 'Pengaturan Skrip Mulai Otomatis',
          description: 'Mengelola skrip yang berjalan secara otomatis saat startup sistem',
          new: 'Baru',
          deleteConfirm: 'Apa kamu yakin menghapus data ini?',
          yes: 'Ya',
          no: 'Tidak',
          scriptName: 'Nama Skrip Mulai Otomatis',
          scriptContent: 'Konten Skrip Mulai Otomatis',
          settings: 'Pengaturan'
        },
        hidOnly: 'HID-Mode Hanya',
        hidOnlyDesc: 'Berhenti meniru perangkat virtual, hanya mempertahankan kontrol dasar HID',
        disk: 'Disk virtual',
        diskDesc: 'Mount virtual U-disk on the remote host',
        rebindNotice:
          'Mengubah sakelar ini akan menghitung ulang perangkat USB, sehingga target sejenak kehilangan perangkat virtual dan jaringan USB-nya.',
        media: {
          title: 'Slot kamera dan mikrofon',
          desc: 'Tetapkan perangkat media yang boleh diisi peramban. Anggaran endpoint diperiksa saat profil USB diterapkan. Menyimpan akan menghitung ulang perangkat dan memutuskan peramban yang terhubung.',
          cameras: 'Kamera',
          microphones: 'Mikrofon',
          name: 'Nama',
          namePlaceholder: 'Ditampilkan pada host tujuan',
          addCamera: 'Tambah kamera',
          addMicrophone: 'Tambah mikrofon',
          remove: 'Hapus',
          cameraDefault: 'Kamera NanoKVM {{index}}',
          microphoneDefault: 'Mikrofon NanoKVM {{index}}',
          nameRequired: 'Setiap slot memerlukan nama.',
          budgetHint:
            'Enam endpoint USB IN adalah batas perangkat keras yang tetap. Satukan papan ketik, tetikus, dan penunjuk absolut pada satu antarmuka HID di Presentasi USB, atau matikan disk virtual di sini atau adaptor jaringan USB di bagian Jaringan.',
          unsupported:
            'Kernel ini tidak dapat menamai perangkat media, sehingga host menampilkan nama bawaan.',
          save: 'Simpan slot',
          disconnect: 'Putuskan',
          disconnectAll: 'Putuskan semua sumber',
          limit: 'Slot kamera dan mikrofon totalnya paling banyak delapan.',
          failed: 'Slot media tidak dapat diperbarui.'
        },
        reboot: 'Mulai ulang',
        rebootDesc: 'Apakah Anda yakin ingin me-reboot NanoKVM?',
        okBtn: 'Ya',
        cancelBtn: 'Tidak'
      },
      network: {
        title: 'Jaringan',
        wifi: {
          title: 'Wi-Fi',
          description: 'Konfigurasi Wi-Fi',
          apMode: 'Mode AP aktif, sambungkan ke Wi-Fi dengan memindai kode QR',
          connect: 'Hubungkan Wi-Fi',
          connectDesc1: 'Masukkan SSID jaringan dan kata sandi',
          connectDesc2: 'Masukkan kata sandi untuk bergabung ke jaringan ini',
          disconnect: 'Yakin ingin memutuskan jaringan?',
          failed: 'Koneksi gagal, coba lagi.',
          ssid: 'Nama',
          password: 'Kata sandi',
          joinBtn: 'Gabung',
          confirmBtn: 'OK',
          cancelBtn: 'Batal'
        },
        tls: {
          description: 'Aktifkan protokol HTTPS',
          tip: 'Perhatian: Menggunakan HTTPS dapat meningkatkan latensi, terutama pada mode video MJPEG.'
        },
        usb: {
          title: 'Adaptor jaringan USB',
          desc: 'Memberi komputer yang dikendalikan kartu jaringan lewat USB',
          type: 'Jenis adaptor',
          typeDesc: 'NCM untuk sistem modern, RNDIS untuk Windows lama'
        },
        bridge: {
          title: 'Adaptor terhubung ke',
          lan: 'Jaringan Anda',
          kvmOnly: 'Hanya NanoKVM',
          lanDesc:
            'Komputer bergabung ke jaringan Anda lewat port Ethernet NanoKVM, dengan alamatnya sendiri dari router.',
          kvmOnlyDesc:
            'Komputer mendapat alamat dari NanoKVM dan bisa menjangkau NanoKVM, tetapi tidak lebih jauh.',
          loading: 'Memuat...',
          state: 'Status',
          states: {
            disabled: 'Hanya NanoKVM',
            enabled: 'Jaringan Anda',
            rolledBack: 'Dikembalikan',
            failed: 'Gagal',
            pending: 'Sedang berjalan'
          },
          uplink: 'Uplink',
          ports: 'Port',
          up: 'aktif',
          down: 'nonaktif',
          noLink: 'tanpa link',
          enableTitle: 'Hubungkan komputer ke jaringan Anda?',
          disableTitle: 'Batasi komputer hanya ke NanoKVM?',
          reconnect:
            'Koneksi manajemen akan terputus sebentar lalu tersambung kembali saat alamat dipindahkan.',
          rollback: 'Jika verifikasi gagal, konfigurasi sebelumnya dipulihkan secara otomatis.',
          enableBtn: 'Gabung ke jaringan saya',
          disableBtn: 'Hanya NanoKVM',
          cancelBtn: 'Batal',
          interrupted:
            'Koneksi terputus saat menerapkan perubahan. Memeriksa ulang status saat ini.',
          pendingNotice: 'Perubahan jembatan masih berjalan atau terhenti sebelum selesai.',
          revert: 'Pulihkan konfigurasi sebelumnya',
          rolledBackNotice:
            'Perubahan terakhir dibatalkan dan konfigurasi sebelumnya telah dipulihkan.',
          verifyFailed: 'Verifikasi gagal: {{gates}}',
          gates: {
            address: 'alamat',
            gateway: 'gerbang',
            inbound: 'koneksi masuk'
          },
          inboundWeak:
            'Pemeriksaan masuk lolos hanya karena NanoKVM menghubungi dirinya sendiri. Itu membuktikan layanan web sedang mendengarkan dan dapat dijangkau secara lokal, bukan bahwa permintaan dari jaringan sampai kepadanya.',
          noCarrier:
            'Tidak ada link di {{port}}. Jembatan tidak punya jalur ke jaringan sampai sebuah kabel terpasang.',
          loop: 'Router juga dipelajari di {{port}}, jadi port itu adalah jalur kedua ke jaringan yang sama. Spanning tree mati, sehingga tidak ada yang akan memutus loop ini: cabut salah satu dari dua jalur tersebut.',
          failedNotice:
            'Perubahan terakhir tidak dapat dibatalkan. NanoKVM mungkin hanya dapat dijangkau melalui AP Wi-Fi atau konsol serial.'
        },
        dns: {
          title: 'DNS',
          description: 'Konfigurasi server DNS untuk NanoKVM',
          mode: 'Mode',
          dhcp: 'DHCP',
          manual: 'Manual',
          add: 'Tambah DNS',
          save: 'Simpan',
          invalid: 'Masukkan alamat IP yang valid',
          noDhcp: 'DNS DHCP saat ini tidak tersedia',
          saved: 'Pengaturan DNS disimpan',
          saveFailed: 'Gagal menyimpan pengaturan DNS',
          unsaved: 'Perubahan belum disimpan',
          maxServers: 'Maksimal {{count}} server DNS diizinkan',
          dnsServers: 'Server DNS',
          dhcpServersDescription: 'Server DNS diperoleh otomatis dari DHCP',
          manualServersDescription: 'Server DNS dapat diedit secara manual',
          networkDetails: 'Detail Jaringan',
          interface: 'Antarmuka',
          ipAddress: 'Alamat IP',
          subnetMask: 'Subnet mask',
          router: 'Router',
          none: 'Tidak ada'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'Server VNC',
        description:
          'Biarkan klien VNC mana pun melihat layar jarak jauh serta memakai papan ketik dan tetikus, dengan masuk memakai akun NanoKVM Anda',
        port: 'Port',
        portDescription: 'Sambungkan ke port ini pada alamat NanoKVM'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Optimasi memori',
          tip: "When memory usage exceeds the limit, garbage collection is performed more aggressively to attempt to free up memory. it's recommended to set to 50MB if using Tailscale. A Tailscale restart is required for the change to take effect."
        },
        swap: {
          title: 'Tukar memori',
          tip: 'Jika masalah terus berlanjut setelah mengaktifkan pengoptimalan memori, coba aktifkan memori swap. Ini menetapkan ukuran file swap ke 256MB secara default, yang dapat disesuaikan di "Pengaturan > Perangkat".'
        },
        restart: 'Are you sure to restart Tailscale?',
        stop: 'Are you sure to stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable its automatic startup on boot.',
        loading: 'Memuat...',
        notInstall: 'Tailscale tidak ditemukan! Silahkan pasang.',
        install: 'Memasang',
        installing: 'Memasangkan',
        failed: 'Gagal memasangkan',
        retry: 'Harap segarkan dan coba lagi. Atau coba instal secara manual',
        download: 'Mengunduh',
        package: 'paket instalasi',
        unzip: 'dan unzip itu',
        upTailscale: 'Unggah tailscale ke direktori NanoKVM /usr/bin/',
        upTailscaled: 'Unggah tailscaled ke direktori NanoKVM /usr/sbin/',
        refresh: 'Segarkan halaman ini',
        notRunning: 'Tailscale tidak berjalan. Silakan mulai untuk melanjutkan.',
        run: 'Mulai',
        notLogin:
          'Perangkat belum ditautkan. Silakan masuk dan tautkan perangkat ini ke akun Anda.',
        urlPeriod: 'Url ini berlaku selama 10 menit',
        login: 'Masuk',
        loginSuccess: 'Berhasil masuk',
        enable: 'Aktifkan Tailscale',
        deviceName: 'Nama Perangkat',
        deviceIP: 'IP Perangkat',
        account: 'Akun',
        logout: 'Keluar',
        logoutDesc: 'Apakah Anda yakin ingin logout?',
        uninstall: 'Copot pemasangan Tailscale',
        uninstallDesc: 'Apakah Anda yakin ingin menghapus instalan Tailscale?',
        okBtn: 'Yes',
        cancelBtn: 'No'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: 'Memuat...',
        notInstall: 'Belum terpasang',
        notConfigured: 'Belum dikonfigurasi',
        stopped: 'Berhenti',
        running: 'Berjalan',
        connected: 'Terhubung',
        error: 'Kesalahan',
        atBoot: 'berjalan saat boot',
        notAtBoot: 'tidak berjalan saat boot',
        arguments: 'Argumen',
        argumentsTip: 'Argumen baris perintah yang diberikan ke layanan saat dijalankan.',
        env: 'Variabel lingkungan',
        envKey: 'Nama',
        envValue: 'Nilai',
        envAdd: 'Tambah variabel',
        envRemove: 'Hapus',
        configured: 'Terkonfigurasi',
        save: 'Simpan',
        saved: 'Konfigurasi disimpan',
        start: 'Mulai',
        stop: 'Hentikan',
        restart: 'Mulai ulang',
        logs: 'Log',
        logsEmpty: 'Belum ada log',
        refresh: 'Segarkan',
        binary: 'Berkas biner',
        binaryShipped: 'Bawaan firmware',
        binaryCustom: 'Biner khusus',
        binaryUpload: 'Unggah biner',
        binaryRevert: 'Kembalikan biner bawaan',
        binaryRevertDesc: 'Hapus biner yang diunggah dan kembalikan versi bawaan firmware?',
        serverWarning: 'Server tanpa pembatasan berfungsi sebagai proxy terbuka',
        noHealthSignal:
          'Layanan ini tidak melaporkan sinyal kesehatan, jadi NanoKVM hanya tahu prosesnya berjalan, bukan apakah terowongannya tersambung.',
        memoryWarning:
          'Menjalankan beberapa layanan akses jarak jauh sekaligus dapat menghabiskan memori',
        resources: 'Sumber daya',
        memory: {
          title: 'Batas memori',
          description:
            'Membatasi heap Go milik newt ke {{limit}} MiB mulai dari mulai ulang berikutnya. Ini batasnya sendiri, bukan milik Tailscale; jika mati, bawaan Go yang berlaku, dan GOGC=50 tetap dipakai pada kedua kasus.',
          noRuntime:
            'wstunnel ditulis dengan Rust: tidak ada pengumpul sampah dan tidak ada batas heap untuk disetel, dan utas pekerjanya sudah mengikuti satu-satunya CPU perangkat ini.',
          notApplicable: 'Tidak berlaku'
        },
        swap: {
          title: 'Berkas swap',
          description:
            'Menambahkan berkas swap 256 MB di kartu SD. Berlaku untuk seluruh sistem: swap yang sama melayani Tailscale, server KVM, dan semua hal lain di perangkat ini.'
        },
        okBtn: 'Ya',
        cancelBtn: 'Tidak'
      },
      update: {
        title: 'Periksa pembaruan',
        queryFailed: 'Gagal mendapatkan versi',
        updateFailed: 'Gagal memperbarui, tolong coba lagi.',
        isLatest: 'Kamu sudah menggunakan versi terbaru.',
        rebooting:
          'Memasang kernel baru dan memulai ulang. Ini bisa memakan beberapa menit; jangan matikan daya.',
        kernelUpdate:
          'Pembaruan ini memasang kernel {{version}}. Perangkat akan dimulai ulang dan kembali sendiri ke kernel saat ini jika yang baru tidak berhasil.',
        rolledBack:
          'Kernel {{version}} gagal dijalankan dan perangkat kembali ke kernel sebelumnya.',
        available: 'Ada pembaruan baru. apa kamu mau memperbarui?',
        updating: 'Pembaruan dimulai. Silahkan tunggu...',
        confirm: 'Konfirmasi',
        cancel: 'Batalkan',
        preview: 'Pratinjau Pembaruan',
        previewDesc: 'Dapatkan akses awal ke fitur dan peningkatan baru',
        previewTip:
          'Perlu diketahui bahwa rilis pratinjau mungkin mengandung bug atau fungsi yang tidak lengkap!',
        customServer: {
          title: 'Server Pembaruan Kustom',
          desc: 'Periksa dan unduh pembaruan daring dari server yang ditentukan',
          invalidUrl:
            'Masukkan direktori server HTTP atau HTTPS yang valid tanpa kueri, fragmen, atau latest.json.',
          loadFailed: 'Gagal memuat konfigurasi server pembaruan.',
          saveFailed: 'Gagal menyimpan konfigurasi server pembaruan.',
          saved: 'Konfigurasi server pembaruan telah disimpan.',
          save: 'Simpan',
          confirmTitle: 'Gunakan server pembaruan kustom?',
          confirmDesc:
            'SHA-512 hanya memeriksa bahwa paket cocok dengan manifes yang disediakan oleh server ini. Pemeriksaan ini tidak membuktikan bahwa paket tersebut merupakan rilis resmi NanoKVM. Server yang bermasalah atau berbahaya dapat membuat perangkat tidak dapat digunakan, menyebabkan kehilangan data, atau membahayakan sistem.',
          confirm: 'Tetap Gunakan',
          previewDisabled:
            'Pembaruan Pratinjau tidak tersedia saat server pembaruan kustom diaktifkan.'
        },
        offline: {
          kernelNotice:
            'Paket ini berisi kernel. Kernel ditulis ke slot cadangan dan perangkat dinyalakan ulang untuk mencobanya; jika tidak kembali, perangkat pulih sendiri ke kernel saat ini.',
          kernelConfirm: 'Pasang kernel',
          kernelCancel: 'Batal',
          title: 'Pembaruan Offline',
          desc: 'Perbarui melalui paket instalasi lokal',
          upload: 'Mengunggah',
          checksumPlaceholder: 'Checksum SHA-256 (opsional)',
          invalidChecksum: 'Checksum SHA-256 harus berisi 64 karakter heksadesimal.',
          checksumMismatch: 'Verifikasi SHA-256 gagal. Paket mungkin rusak.',
          invalidName: 'Format nama file tidak valid. Silakan unduh dari rilis GitHub.',
          updateFailed: 'Gagal memperbarui, tolong coba lagi.'
        }
      },
      account: {
        title: 'Akun',
        webAccount: 'Nama akun web',
        role: 'Peran',
        roles: {
          admin: 'Administrator',
          user: 'Pengguna'
        },
        password: 'Kata sandi',
        updateBtn: 'Update',
        logoutBtn: 'Keluar',
        logoutDesc: 'Apakah Anda yakin ingin logout?',
        okBtn: 'Ya',
        cancelBtn: 'Tidak',
        users: {
          title: 'Pengguna',
          create: 'Buat pengguna',
          enabled: 'Aktif',
          disabled: 'Nonaktif',
          deviceOwner: 'Pemilik perangkat',
          resetPassword: 'Atur ulang kata sandi',
          delete: 'Hapus',
          deleteConfirm: 'Hapus pengguna ini dan cabut semua sesinya?',
          created: 'Pengguna dibuat',
          deleted: 'Pengguna dihapus',
          passwordUpdated: 'Kata sandi diperbarui',
          loadFailed: 'Gagal memuat pengguna',
          saveFailed: 'Gagal menyimpan pengguna',
          deleteFailed: 'Gagal menghapus pengguna'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Asisten',
      empty: 'Buka panel dan mulai tugas untuk memulai.',
      inputPlaceholder: 'Jelaskan apa yang Anda ingin PicoClaw lakukan',
      newConversation: 'Percakapan baru',
      processing: 'Memproses...',
      agent: {
        defaultTitle: 'Asisten umum',
        defaultDescription: 'Obrolan umum, pencarian, dan bantuan ruang kerja.',
        kvmTitle: 'Kontrol jarak jauh',
        kvmDescription: 'Operasikan host jarak jauh melalui NanoKVM.',
        switched: 'Peran agen dialihkan',
        switchFailed: 'Gagal mengganti peran agen'
      },
      send: 'Kirim',
      cancel: 'Batalkan',
      status: {
        connecting: 'Menghubungkan ke gerbang...',
        connected: 'Sesi PicoClaw terhubung',
        disconnected: 'Sesi PicoClaw ditutup',
        stopped: 'Permintaan penghentian terkirim',
        runtimeStarted: 'Runtime PicoClaw dimulai',
        runtimeStartFailed: 'Gagal memulai Runtime PicoClaw',
        runtimeStopped: 'Runtime PicoClaw dihentikan',
        runtimeStopFailed: 'Gagal menghentikan Runtime PicoClaw',
        controlSwitchedToMCP: 'Kontrol dialihkan ke layanan MCP eksternal'
      },
      connection: {
        runtime: {
          checking: 'Memeriksa',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime siap',
          stopped: 'Runtime dihentikan',
          blockedByMCP: 'Kontrol MCP eksternal sedang aktif',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime tidak tersedia',
          configError: 'Kesalahan konfigurasi'
        },
        transport: {
          connecting: 'Menghubungkan',
          connected: 'Terhubung',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Menganggur',
          busy: 'Sibuk'
        }
      },
      message: {
        toolAction: 'Aksi',
        observation: 'Pengamatan',
        screenshot: 'Tangkapan layar'
      },
      overlay: {
        locked: 'PicoClaw sedang mengendalikan perangkat. Input manual dijeda.'
      },
      control: {
        picoclaw: 'Kontrol perangkat: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Kontrol perangkat: MCP eksternal',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Kontrol perangkat: nonaktif',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Berikan kontrol',
        release: 'Lepaskan',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'Kontrol PicoClaw diberikan',
        released: 'Kontrol PicoClaw dilepaskan',
        grantFailed: 'Gagal memberikan kontrol PicoClaw',
        releaseFailed: 'Gagal melepaskan kontrol PicoClaw',
        grantConfirmTitle: 'Alihkan kontrol perangkat ke PicoClaw?',
        grantConfirmDesc: 'Penulisan perangkat MCP eksternal akan dihentikan.'
      },
      install: {
        install: 'Instal PicoClaw',
        installing: 'Menginstal PicoClaw',
        success: 'PicoClaw berhasil diinstal',
        failed: 'Gagal menginstal PicoClaw',
        uninstalling: 'Mencopot pemasangan runtime...',
        uninstalled: 'Runtime berhasil di-uninstall.',
        uninstallFailed: 'Pencopotan pemasangan gagal.',
        requiredTitle: 'PicoClaw tidak diinstal',
        requiredDescription: 'Instal PicoClaw sebelum memulai runtime PicoClaw.',
        progressDescription: 'PicoClaw sedang diunduh dan diinstal.',
        stages: {
          preparing: 'Mempersiapkan',
          downloading: 'Mengunduh',
          extracting: 'Mengekstraksi',
          verifying: 'Memverifikasi',
          installing: 'Memasangkan',
          installed: 'Terpasang',
          install_timeout: 'Waktu Habis',
          install_failed: 'Gagal'
        }
      },
      model: {
        requiredTitle: 'Konfigurasi model diperlukan',
        requiredDescription: 'Konfigurasikan model PicoClaw sebelum menggunakan obrolan PicoClaw.',
        docsTitle: 'Panduan Konfigurasi',
        docsDesc: 'Model dan protokol yang didukung',
        menuLabel: 'Konfigurasi model',
        modelIdentifier: 'Pengenal Model',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'Kunci API',
        apiKeyPlaceholder: 'Masukkan kunci API model',
        save: 'Simpan',
        saving: 'Menyimpan',
        saved: 'Konfigurasi model disimpan',
        saveFailed: 'Gagal menyimpan konfigurasi model',
        invalid: 'Pengidentifikasi model, API Base URL, dan kunci API wajib diisi'
      },
      uninstall: {
        menuLabel: 'Copot pemasangan',
        confirmTitle: 'Copot pemasangan PicoClaw',
        confirmContent:
          'Apakah Anda yakin ingin menghapus instalan PicoClaw? Ini akan menghapus semua file yang dapat dieksekusi dan konfigurasi.',
        confirmOk: 'Copot pemasangan',
        confirmCancel: 'Batalkan'
      },
      history: {
        title: 'Riwayat',
        loading: 'Memuat sesi...',
        emptyTitle: 'Belum ada riwayat',
        emptyDescription: 'Sesi PicoClaw sebelumnya akan muncul di sini.',
        loadFailed: 'Gagal memuat riwayat sesi',
        deleteFailed: 'Gagal menghapus sesi',
        deleteConfirmTitle: 'Hapus sesi',
        deleteConfirmContent: 'Apakah Anda yakin ingin menghapus "{{title}}"?',
        deleteConfirmOk: 'Hapus',
        deleteConfirmCancel: 'Batalkan',
        messageCount_one: '{{count}} pesan',
        messageCount_other: '{{count}} pesan',
        messageCount: '{{count}} pesan'
      },
      config: {
        startRuntime: 'Mulai PicoClaw',
        stopRuntime: 'Hentikan PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Alihkan kontrol ke PicoClaw?',
        enableConfirmDesc: 'Memulai PicoClaw akan menonaktifkan layanan MCP eksternal.',
        enableConfirmOk: 'Mulai PicoClaw',
        enableConfirmCancel: 'Batal',
        title: 'Mulai PicoClaw',
        description: 'Mulai runtime untuk mulai menggunakan asisten PicoClaw.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Kami mengalami masalah',
      refresh: 'Segarkan'
    },
    fullscreen: {
      toggle: 'Beralih Layar Penuh'
    },
    menu: {
      collapse: 'Tutup Menu',
      expand: 'Perluas Menu'
    }
  }
};

export default id;
