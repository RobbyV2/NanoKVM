const pl = {
  translation: {
    head: {
      desktop: 'Zdalny pulpit',
      login: 'Logowanie',
      changePassword: 'Zmień Hasło',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Logowanie',
      placeholderUsername: 'Wprowadź nazwę użykownika',
      placeholderPassword: 'wprowadź hasło',
      placeholderCurrentPassword: 'Bieżące hasło',
      placeholderPassword2: 'wprowadź hasło ponownie',
      noEmptyUsername: 'nazwa użykownika nie może być pusta',
      noEmptyPassword: 'hasło nie może być puste',
      passwordLength: 'Hasło musi mieć od 8 do 72 znaków',
      noAccount:
        'Nie udało się uzyskać informacji o użytkowniku, odśwież stronę lub zresetuj hasło',
      invalidUser: 'Błędne hasło lub nazwa użykownika',
      locked: 'Zbyt wiele loginów, spróbuj ponownie później',
      globalLocked: 'System chroniony, spróbuj ponownie później',
      error: 'niespodziewany błąd',
      invalidCurrentPassword: 'Bieżące hasło jest nieprawidłowe',
      changePassword: 'Zmień Hasło',
      changePasswordDesc:
        'Dla bezpieczeństwa Twojego urządzenia, proszę zmień hasło do logowania w sieci.',
      differentPassword: 'hasła nie zgadzają się',
      illegalUsername: 'nazwa użytkownika zawiera niedozwolone znaki',
      illegalPassword: 'hasło zawiera niedozwolone znaki',
      forgetPassword: 'Zapomiałeś hasła?',
      ok: 'Ok',
      cancel: 'Anuluj',
      loginButtonText: 'Zaloguj się',
      tips: {
        reset1:
          'To reset the passwords, pressing and holding the BOOT button on the NanoKVM for 10 seconds.',
        reset2: 'Szczegółowe kroki znajdziesz w tym dokumencie:',
        reset3: 'Domyślne konto web:',
        reset4: 'Domyślne konto SSH:',
        change1: 'Pamiętaj, że ta operacja zmieni następujące hasła:',
        change2: 'Hasło logowania web',
        change3: 'Hasło roota systemu (hasło logowania SSH)',
        change4: 'Aby zresetować hasła, naciśnij i przytrzymaj przycisk BOOT na NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Skonfiguruj Wi-Fi dla NanoKVM',
      success: 'Proszę podejść do urządzenia, aby sprawdzić stan sieci NanoKVM.',
      failed: 'Operacja nie powiodła się, spróbuj ponownie.',
      invalidMode:
        'Bieżący tryb nie obsługuje konfiguracji sieci. Przejdź do swojego urządzenia i włącz tryb konfiguracji Wi-Fi.',
      confirmBtn: 'Ok',
      finishBtn: 'Zakończono',
      ap: {
        authTitle: 'Wymagane uwierzytelnienie',
        authDescription: 'Aby kontynuować, wprowadź AP hasło',
        authFailed: 'Nieprawidłowe hasło AP',
        passPlaceholder: 'AP hasło',
        verifyBtn: 'Sprawdź'
      }
    },
    screen: {
      scale: 'Skala',
      title: 'Ekran',
      video: 'Tryb wideo',
      videoDirectTips: 'Włącz HTTPS w „Ustawienia > Urządzenie”, aby korzystać z tego trybu',
      resolution: 'Rozdzielczość',
      auto: 'Automatyczny',
      autoTips:
        'W określonych rozdzielczościach może wystąpić rozrywanie ekranu lub przesunięcie myszy. Rozważ dostosowanie rozdzielczości zdalnego hosta lub wyłącz tryb automatyczny.',
      fps: 'FPS',
      customizeFps: 'Personalizuj',
      quality: 'Jakość',
      qualityLossless: 'Bezstratny',
      qualityHigh: 'Wysoki',
      qualityMedium: 'Średni',
      qualityLow: 'Niski',
      frameDetect: 'Wykrywanie klatek',
      frameDetectTip:
        'Obliczanie różnicy między klatkami. Zatrzymaj transmisję strumienia wideo, gdy na ekranie zdalnego hosta nie zostaną wykryte żadne zmiany.',
      resetHdmi: 'Resetuj HDMI',
      mixedH264: {
        title: 'Konflikt strumieni H.264',
        description:
          'Strumienie H.264 Direct i H.264 WebRTC są używane jednocześnie. Może to powodować rozrywanie obrazu lub uszkodzenie wideo. Używaj tylko jednego trybu H.264.'
      },
      webrtcConnectionFailed: {
        title: 'Połączenie WebRTC nie powiodło się',
        description: 'Sprawdź połączenie sieciowe lub zmień tryb wideo.'
      },
      captureStatus: {
        hdmiError: 'Błąd obrazu HDMI',
        unsupportedResolution: 'Bieżąca rozdzielczość nie jest obsługiwana',
        retrieving: 'Pobieranie obrazu...',
        changingResolution: 'Przełączanie rozdzielczości...',
        updateFailed: 'Nie można teraz zaktualizować obrazu',
        videoError: 'Błąd wyświetlania wideo',
        noHdmi: 'Nie wykryto sygnału HDMI',
        unavailable: 'Nie można teraz wyświetlić obrazu'
      }
    },
    keyboard: {
      title: 'Klawiatura',
      paste: 'Wklej',
      tips: 'Tylko standardowe klawiaturowe znaki i symbole są obsługiwane.',
      placeholder: 'Proszę wprowadzić coś',
      submit: 'Prześlij',
      virtual: 'Klawiatura',
      readClipboard: 'Czytaj ze schowka',
      clipboardPermissionDenied:
        'Odmowa dostępu do schowka. Zezwól na dostęp do schowka w przeglądarce.',
      clipboardReadError: 'Nie udało się odczytać schowka',
      dropdownEnglish: 'Angielski',
      dropdownGerman: 'niemiecki',
      dropdownFrench: 'Francuski',
      dropdownRussian: 'Rosyjski',
      shortcut: {
        title: 'Skróty',
        custom: 'Niestandardowe',
        capture: 'Kliknij tutaj, aby przechwycić skrót',
        clear: 'Jasne',
        save: 'Zapisz',
        captureTips:
          'Przechwytywanie klawiszy systemowych (takich jak klawisz Windows) wymaga uprawnienia do pełnego ekranu.',
        enterFullScreen: 'Przełącz tryb pełnoekranowy.'
      },
      leaderKey: {
        title: 'Klawisz Leader',
        desc: 'Omiń ograniczenia przeglądarki i wyślij skróty systemowe bezpośrednio do zdalnego hosta.',
        howToUse: 'Jak używać',
        simultaneous: {
          title: 'Tryb symultaniczny',
          desc1: 'Naciśnij i przytrzymaj klawisz Leader, a następnie naciśnij skrót.',
          desc2: 'Intuicyjne, ale może kolidować ze skrótami systemowymi.'
        },
        sequential: {
          title: 'Tryb sekwencyjny',
          desc1:
            'Naciśnij klawisz Leader → naciśnij skrót w sekwencji → ponownie naciśnij klawisz Leader.',
          desc2:
            'Wymaga większej liczby kroków, ale całkowicie pozwala uniknąć konfliktów systemowych.'
        },
        enable: 'Włącz klawisz Leader',
        tip: 'Po przypisaniu jako klawisz Leader ten klawisz działa wyłącznie jako wyzwalacz skrótów i traci swoje domyślne zachowanie.',
        placeholder: 'Naciśnij klawisz Leader',
        shiftRight: 'Prawy Shift',
        ctrlRight: 'Prawy Ctrl',
        metaRight: 'Prawy Win',
        submit: 'Prześlij',
        recorder: {
          rec: 'NAGR',
          activate: 'Aktywuj klawisze',
          input: 'Proszę nacisnąć skrót...'
        }
      }
    },
    mouse: {
      title: 'Mysz',
      cursor: 'Styl kursora',
      default: 'Domyślny kursor',
      pointer: 'Wskazujący kursor',
      cell: 'Kursor komórki',
      text: 'Kursor tekstowy',
      grab: 'Kursor chwytania',
      hide: 'Ukruj kursor',
      mode: 'Tryb myszki',
      absolute: 'Tryb bezwzględny',
      relative: 'Tryb względny',
      direction: 'Kierunek kółka przewijania',
      scrollUp: 'Przewiń w górę',
      scrollDown: 'Przewiń w dół',
      speed: 'Szybkość kółka przewijania',
      fast: 'Szybko',
      slow: 'Powoli',
      requestPointer: 'Korzystanie z trybu względnego. Kliknij pulpit, aby uzyskać wskaźnik myszy.',
      resetHid: 'Zresetuj HID',
      hidOnly: {
        title: 'Tryb tylko HID',
        desc: 'Jeśli mysz i klawiatura przestaną odpowiadać, a resetowanie HID nie pomoże, może to oznaczać problem ze zgodnością między NanoKVM a urządzeniem. Spróbuj włączyć tryb HID-Only, aby uzyskać lepszą kompatybilność.',
        tip1: 'Włączenie trybu HID-Only spowoduje odmontowanie wirtualnego dysku U i sieci wirtualnej',
        tip2: 'W trybie HID-Only montowanie obrazu jest wyłączone',
        tip3: 'NanoKVM automatycznie uruchomi się ponownie po przełączeniu trybów',
        enable: 'Włącz tryb HID-Only',
        disable: 'Wyłącz tryb HID-Tylko'
      }
    },
    image: {
      title: 'Obrazy',
      loading: 'Ładowanie...',
      empty: 'Nic nie znaleziono',
      mountMode: 'Tryb montowania',
      mountFailed: 'Nie udało się zamontować obrazu',
      mountDesc:
        'W niektórych systemach wymagane jest wyjęcie dysku wirtualnego na zdalnym hoście przed zamontowaniem obrazu.',
      unmountFailed: 'Odmontowanie nie powiodło się',
      unmountDesc:
        'W niektórych systemach należy ręcznie wysunąć obraz ze zdalnego hosta przed odmontowaniem obrazu.',
      refresh: 'Odśwież listę obrazów',
      attention: 'Uwaga',
      deleteConfirm: 'Czy na pewno chcesz usunąć to zdjęcie?',
      okBtn: 'Tak',
      cancelBtn: 'Nie',
      tips: {
        title: 'Jak przesłać obrazy',
        usb1: 'Podłącz urządzenie NanoKVM do komputera przez USB.',
        usb2: 'Upewnij się, że dysk wirtualny jest zamontowany (Ustawienia - Dysk wirtualny).',
        usb3: 'Otwórz dysk wirtualny na swoim komputerze i skopiuj plik obrazu do katalogu głównego dysku wirtualnego.',
        scp1: 'Upewnij się że NanoKVM i twój komputer są na tej samej sieci lokalnej.',
        scp2: 'Otwórz terminal na komputerze i użyj komendę SCP aby przesłać obraz do katalogu /data na NanoKVM.',
        scp3: 'Przykład: scp lokalizacja-zrodlowego-obrazu root@ip-twojego-nanokvm:/data',
        tfCard: 'Karta SD',
        tf1: 'Ta metoda jest obsługiwana w systemie Linux',
        tf2: 'Usuń kartę SD od NanoKVM (dla wersji FULL, rozbierz obudowę najpierw).',
        tf3: 'Włóż kartę SD do czytnika kart i podłącz do twojego komputera.',
        tf4: 'Kopjuj obraz do katalogu /data na karcie SD.',
        tf5: 'Włóż kartę SD do NanoKVM.'
      }
    },
    script: {
      title: 'Skrypty',
      upload: 'Prześlij',
      run: 'Uruchom',
      runBackground: 'Uruchomiony w tle',
      runFailed: 'Uruchomienie nie powiodło się',
      attention: 'Uwaga',
      delDesc: 'Czy na pewno chcesz usunąć ten plik?',
      confirm: 'Tak',
      cancel: 'Nie',
      delete: 'Usuń',
      close: 'Zamknij'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal NanoKVM',
      serial: 'Terminal portu szeregowego',
      serialPort: 'Port szeregowy',
      serialPortPlaceholder: 'Wprowadź port szeregowy',
      baudrate: 'Szybkość transmisji',
      parity: 'Kontrola parzystości',
      parityNone: 'Brak kontroli',
      parityEven: 'Parzysta',
      parityOdd: 'Nieparzysta',
      flowControl: 'Kontrola przepływu',
      flowControlNone: 'Brak kontroli',
      flowControlSoft: 'Programowe',
      flowControlHard: 'Sprzętowe',
      dataBits: 'Bity danych',
      stopBits: 'Bity stopu',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Wysyłanie komendy...',
      sent: 'Komenda wysłana',
      input: 'Wprowadź numer adresu MAC',
      ok: 'Ok'
    },
    download: {
      title: 'Narzędzie do pobierania obrazów',
      input: 'Proszę wprowadzić zdalny obraz URL',
      ok: 'Ok',
      disabled: '/data partycja to RO, więc nie możemy pobrać obrazu',
      uploadbox: 'Upuść plik tutaj lub kliknij, aby wybrać',
      inputfile: 'Proszę wprowadzić plik obrazu',
      NoISO: 'Brak ISO',
      sha256: 'SHA-256 (opcjonalnie)',
      sha256Placeholder: 'Wprowadź 64-znakową sumę kontrolną SHA-256',
      invalidSHA256: 'SHA-256 musi być 64-znakowym ciągiem szesnastkowym',
      failed: 'Pobieranie nie powiodło się',
      success: 'Pobieranie zakończone pomyślnie',
      checksumFailed: 'Pobieranie nie powiodło się: weryfikacja SHA-256 nie powiodła się',
      cancel: 'Anuluj',
      cancelFailed: 'Nie udało się anulować pobierania'
    },
    power: {
      title: 'Zasilanie',
      showConfirm: 'Potwierdzenie',
      showConfirmTip: 'Operacje zasilania wymagają dodatkowego potwierdzenia',
      reset: 'Resetuj',
      power: 'Zasilanie',
      powerShort: 'Zasilanie (krótkie kliknięcie)',
      powerLong: 'Zasilanie (długie kliknięcie)',
      resetConfirm: 'Kontynuować operację resetowania?',
      powerConfirm: 'Kontynuować zasilanie?',
      okBtn: 'Tak',
      cancelBtn: 'Nie'
    },
    devices: {
      title: 'Urządzenia',
      stale: 'Stan urządzeń na żywo jest niedostępny. Trwa ponowne łączenie.',
      empty: 'Nie skonfigurowano gniazd kamery ani mikrofonu. Dodaj je w Ustawieniach, Urządzenie.',
      available: 'Dostępny',
      waiting: 'Host czeka na źródło',
      hostOpen: 'Host otwarty',
      hostIdle: 'Host bezczynny',
      sending: 'Nadawanie z tej przeglądarki',
      black: 'Czarny obraz',
      silence: 'Cyfrowa cisza',
      resuming: 'Oczekiwanie na wznowienie',
      stop: 'Zatrzymaj udostępnianie',
      disconnect: 'Odłącz',
      takeover: 'Przejmij',
      refused: 'Używane przez {{owner}} ze źródła {{source}}',
      connectedSources_one: '{{count}} podłączone źródło',
      connectedSources_other: '{{count}} podłączonych źródeł',
      connectedSources: '{{count}} podłączonych źródeł',
      connection: {
        connecting: 'Łączenie',
        connected: 'Na żywo',
        disconnected: 'Ponowne łączenie'
      },
      share: {
        camera: 'Udostępnij kamerę',
        microphone: 'Udostępnij mikrofon',
        usbDevice: 'Udostępnij USB'
      },
      permission: {
        denied: 'Zablokowane w ustawieniach witryny w przeglądarce',
        prompt: 'Przeglądarka poprosi o dostęp'
      },
      mic: {
        mute: 'Wycisz',
        unmute: 'Wyłącz wyciszenie'
      },
      revoked: {
        released: 'Udostępnianie zostało zatrzymane',
        lease_expired: 'Dzierżawa wygasła, zanim ta przeglądarka wróciła',
        admin_disconnect: 'Administrator odłączył wszystkie źródła',
        slot_removed: 'Slot został usunięty',
        slot_changed: 'Slot został zmieniony',
        taken_over: 'Administrator przejął ten slot'
      },
      usb: {
        surrendered: 'Passthrough USB trzyma klawiaturę i mysz',
        surrenderedDesc:
          'Zdalny host widzi zaimportowane urządzenie zamiast klawiatury, myszy i nośników wirtualnych NanoKVM. Wracają, gdy sesja się kończy.',
        unsupported: 'WebUSB wymaga przeglądarki opartej na Chromium',
        insecure: 'Ta strona nie jest serwowana przez HTTPS, więc przeglądarka blokuje WebUSB. Włącz HTTPS w Ustawieniach, Sieć.',
        session: 'Przekazywanie {{device}} ({{mode}})',
        idle: 'Brak sesji passthrough',
        mode: {
          hybrid: 'hybrydowy',
          exact: 'dokładny'
        }
      }
    },
    settings: {
      title: 'Ustawienia',
      display: {
        title: 'Ekran',
        loading: 'Ładowanie...',
        active: 'Aktywny EDID',
        activeUnknown:
          'NanoKVM nie zapisał żadnego EDID od uruchomienia, więc tożsamość widziana przez hosta jest nieznana.',
        appliedAt: 'Zastosowano {{time}}',
        download: 'Pobierz',
        downloadBackup: 'Pobierz poprzedni',
        preset: 'Profil monitora',
        presetPlaceholder: 'Wybierz monitor',
        upload: 'Wyślij',
        selected: 'Wybrany EDID',
        errors: 'Błędy',
        warnings: 'Ostrzeżenia',
        info: 'Informacje',
        unknownMonitor: 'Nieznany monitor',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Dźwięk',
        audioNo: 'Bez dźwięku',
        extensionBlocks: 'Bloki rozszerzeń: {{blocks}}',
        apply: 'Zastosuj',
        applyTitle: 'Zastosować ten EDID?',
        before: 'Obecny',
        after: 'Nowy',
        hdmiNotice:
          'Podczas zapisu EDID przechwytywanie obrazu zatrzymuje się i uruchamia ponownie samo.',
        powerCycleNotice:
          'To urządzenie trzeba fizycznie odłączyć od zasilania i podłączyć ponownie, aby nowy EDID zaczął działać.',
        powerCycleUnverified:
          'Zapisu nie udało się zweryfikować, więc układ wideo zachowa to, co ma teraz, dopóki tego urządzenia nie odłączysz fizycznie od zasilania i nie podłączysz ponownie.',
        applied: 'EDID zastosowany i zweryfikowany.',
        applyFailed: 'Zastosowanie EDID nie powiodło się.',
        busy: 'Układ wideo był zajęty. Spróbuj ponownie.',
        unsupported: 'To urządzenie nie obsługuje zmiany EDID.',
        toolMissing: 'W tym oprogramowaniu brakuje narzędzia EDID.',
        noAudio: 'Ten EDID nie zgłasza dźwięku, więc host może przestać go wysyłać.',
        oldVersion: 'Ten EDID używa wersji starszej niż 1.4.',
        interlaced: 'Preferowane taktowanie jest z przeplotem.',
        tooLarge:
          'Preferowane taktowanie przekracza 1920x1080 przy 60 Hz, czyli więcej, niż NanoKVM może przechwycić.',
        recovery: 'Odzyskiwanie',
        recoveryNeeded:
          'Ostatni zapis nie został zweryfikowany, więc obszar EDID w układzie wideo jest w nieznanym stanie. Przywróć fabryczny EDID, aby stan znów był znany.',
        recoveryDesc:
          'Zapisz znany EDID z powrotem w układzie wideo, jeśli zastosowany EDID pozostawił hosta bez obrazu.',
        restoreFactory: 'Przywróć fabryczny EDID',
        restoreBackup: 'Przywróć poprzedni EDID',
        restoreTitle: 'Przywrócić ten EDID?',
        restoreFactoryTarget: 'Fabryczny EDID dostarczany z NanoKVM.',
        restoreBackupTarget: 'Najnowsza kopia zapasowa, zastosowana {{time}}.',
        restoreNotice:
          'Przywracanie jest zapisywane tak samo jak zastosowanie i ma te same konsekwencje.',
        restored: 'EDID przywrócony i zweryfikowany.',
        restoreFailed: 'Przywracanie EDID nie powiodło się.',
        mismatchTitle: 'Zapisane i odczytane z powrotem',
        mismatchWritten: 'Zapisane',
        mismatchRead: 'Odczytane z powrotem',
        restoreOkBtn: 'Przywróć',
        hardware: 'Wykryty sprzęt: {{hardware}}',
        hardwareUnknown: 'Nieznany',
        confirmWord: 'ZASTOSUJ',
        confirmPrompt: 'Wpisz {{word}}, aby odblokować przycisk zastosowania.',
        okBtn: 'Zastosuj',
        cancelBtn: 'Anuluj'
      },
      presentation: {
        title: 'Prezentacja USB',
        loading: 'Ładowanie...',
        current: 'Bieżąca prezentacja USB',
        noProfile: 'Nie zastosowano żadnego profilu',
        linked: 'Powiązane funkcje',
        hostState: 'USB hosta',
        hostUnbound: 'Kontroler niepowiązany',
        hdmiState: 'Wejście HDMI',
        hdmiSignal: 'Sygnał obecny',
        hdmiUnreported: 'Brak jeszcze raportu przechwytywania',
        endpoints: 'Endpointy',
        fifos: 'Sloty FIFO',
        pending: 'Oczekujące zmiany',
        pendingEdits: 'Niezapisane zmiany tożsamości',
        pendingProfile: '{{profile}} jest wybrany, ale nie zastosowany',
        pendingNone: 'Brak',
        lastApply: 'Ostatnie zastosowanie',
        applyFailed: 'Niepowodzenie na {{profile}} o {{time}}',
        applyClean: 'Nie zapisano żadnego błędu',
        lastKnownGood: 'Ostatni znany działający',
        rollbackTarget: 'Cel wycofania',
        rollbackNone: 'Brak',
        powerCyclePending:
          'Kontroler został zabrany hostowi. Wyłącz i włącz ponownie podłączony komputer, aby odzyskać urządzenie.',
        rollback: 'Wycofaj',
        rollbackTitle: 'Wycofać do {{profile}}?',
        rollbackDesc: 'Gadżet zostanie ponownie wyliczony; funkcje USB na chwilę znikną.',
        profile: 'Profil USB',
        builtIn: 'wbudowany',
        descriptors: 'deskryptory',
        imported: 'zaimportowany',
        clone: 'Sklonuj',
        cloneTitle: 'Sklonuj ten profil',
        cloneToEdit:
          'Wbudowane profile pozostają tylko do odczytu. Sklonuj ten profil, aby edytować jego tożsamość.',
        profileName: 'Nazwa profilu',
        profileNameHint: 'Małe litery, cyfry, kropki, podkreślenia i myślniki.',
        import: 'Importuj pakiet',
        export: 'Eksportuj pakiet',
        delete: 'Usuń',
        deleteTitle: 'Usunąć ten profil?',
        deleteDesc: 'Usuń {{profile}} z NanoKVM.',
        identity: 'Tożsamość USB',
        preset: 'Predefiniowana tożsamość',
        presetPlaceholder: 'Skopiuj tożsamość ze znanego urządzenia',
        presetHint:
          'Predefiniowana tożsamość wypełnia Vendor ID, Product ID i oba pola nazwy. Nie niesie ze sobą żadnych deskryptorów.',
        presetSource: 'Tożsamość zapisana w {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'To Vendor ID należy do innego producenta',
        productId: 'Product ID',
        bcdUSB: 'Wersja USB',
        bcdDevice: 'Wersja urządzenia',
        manufacturer: 'Producent',
        product: 'Produkt',
        serial: 'Numer seryjny',
        configuration: 'Ciąg konfiguracji',
        hidLayout: 'Urządzenia HID',
        hidRoleKeyboard: 'Klawiatura',
        hidRoleRelative: 'Mysz (względna)',
        hidRoleAbsolute: 'Wskaźnik (bezwzględny)',
        hidOff: 'Brak',
        hidInterface: 'Interfejs {{index}}',
        hidBootKeyboardShared:
          'Klawiatura współdzieli interfejs, więc nie udostępnia już raportu w protokole boot. Część BIOS-ów i UEFI jej nie zobaczy.',
        functions: 'Funkcje',
        descriptorAssets: 'Zapisane pliki deskryptorów: {{count}}',
        endpointUse:
          'IN {{inUse}} zajęte, {{inFree}} wolne; OUT {{outUse}} zajęte, {{outFree}} wolne',
        preview: 'Sprawdź',
        save: 'Zapisz',
        apply: 'Zastosuj',
        applyTitle: 'Zastosować ten profil USB?',
        applyDesc: 'NanoKVM przedstawi podłączonemu komputerowi {{profile}}.',
        reconnect:
          'Klawiatura, mysz i pozostałe funkcje USB na chwilę się rozłączą, gdy gadżet zostanie powiązany na nowo.',
        applyLinks: 'Powiąże: {{functions}}',
        applyRemoves: 'Usunie: {{functions}}',
        applyNoHid:
          'Po tym zastosowaniu nie zostanie żadna funkcja HID. Klawiatura i mysz przestaną działać.',
        applyRollback: 'Nieudane zastosowanie wróci do {{profile}}.',
        recoveryPowerCycle:
          'Żadne HID nie przetrwa tego zastosowania, więc hosta, który przestanie odpowiadać, da się odzyskać tylko przez wyłączenie i włączenie zasilania.',
        recoveryReboot:
          'Z urządzenia złożonego zniknie jeden interfejs; host może potrzebować ponownego uruchomienia, aby powiązać resztę.',
        recoveryHdmiReset:
          'Funkcja wideo jest budowana od nowa, więc stojący za nią tor przechwytywania zostaje zresetowany.',
        recoveryReconnect: 'Host ponownie wylicza urządzenie; funkcje USB na chwilę znikną.',
        cancel: 'Anuluj',
        noFunctions: 'Brak powiązanych funkcji',
        loadFailed: 'Nie udało się wczytać profili prezentacji',
        operationFailed: 'Operacja na prezentacji nie powiodła się'
      },
      passthrough: {
        title: 'Przekazywanie USB',
        loading: 'Wczytywanie...',
        mode: 'Tryb',
        hybrid: 'Hybrydowy',
        exact: 'Dokładny',
        hybridDesc: 'Zachowuje klawiaturę boot i mysz względną, dla zgodnych urządzeń.',
        exactDesc: 'Zastępuje każdą funkcję USB NanoKVM przekazanym urządzeniem.',
        hybridWarning: 'Tryb hybrydowy pozostawia klawiaturę i mysz względną dostępne',
        hybridWarningDesc:
          'Pamięć masowa, sieć po USB i wskaźnik bezwzględny rozłączają się na czas działania przekazanej funkcji.',
        hidWarning: 'Uruchomienie przekazywania oddaje klawiaturę, mysz i nośniki wirtualne',
        hidWarningDesc:
          'NanoKVM ma tylko jeden kontroler urządzenia USB, a proxy potrzebuje go w całości. Dlatego w trakcie sesji zdalny host widzi przekazane urządzenie zamiast klawiatury, myszy i nośników wirtualnych NanoKVM. Wracają samoczynnie w chwili zatrzymania sesji. Ten interfejs webowy działa niezależnie, więc sesję zawsze można zatrzymać z tej strony.',
        hidWarningSafeDesc:
          'NanoKVM ma tylko jeden kontroler urządzenia USB, a proxy potrzebuje go w całości. Dlatego w trakcie sesji zdalny host widzi przekazane urządzenie zamiast klawiatury, myszy i nośników wirtualnych NanoKVM. Wracają po zatrzymaniu sesji.',
        isoLabel: 'Zezwól na transfery izochroniczne',
        isoHint:
          'Przepuszcza kamery internetowe, mikrofony i inne urządzenia strumieniowe. Nikt nie zmierzył, ile ten sprzęt udźwignie.',
        isoWarning:
          'Strumień izochroniczny nie jest tu sprawdzony i może zatrzymać klawiaturę i mysz do czasu zakończenia sesji',
        info: {
          title: 'Informacje',
          hybrid:
            'Tryb hybrydowy pozostawia klawiaturę i mysz względną dostępne. Pamięć masowa, sieć po USB i wskaźnik bezwzględny rozłączają się na czas działania przekazanego urządzenia.',
          exact:
            'Tryb dokładny zastępuje każdą funkcję USB NanoKVM przekazanym urządzeniem. Klawiatura, mysz i nośniki wirtualne wracają samoczynnie po zatrzymaniu sesji.',
          udc: 'NanoKVM ma tylko jeden kontroler urządzenia USB, a proxy potrzebuje go w całości — dlatego powyższe funkcje znikają na czas trwania sesji.',
          web: 'Ten interfejs webowy działa niezależnie, więc sesję zawsze można zatrzymać z tej strony.',
          network:
            'Przekazywanie uruchamiaj przez Ethernet lub Wi-Fi. Uruchomienie z sieci USB NanoKVM jest odrzucane, bo to połączenie by zniknęło.',
          iso: 'Kamery internetowe, mikrofony i inne urządzenia izochroniczne są odrzucane, dopóki nie zezwolisz na transfery izochroniczne. Ta ścieżka działa, ale nigdy nie zmierzono jej na tym sprzęcie, więc traktuj przepustowość jako nieznaną.',
          camera:
            'Kamera i mikrofon przeglądarki w sekcji Urządzenia pozostają sprawdzonym sposobem udostępnienia ich hostowi.'
        },
        session: 'Sesja',
        activeDesc: 'Urządzenie jest zaimportowane, a proxy trzyma kontroler USB.',
        inactiveDesc:
          'Żadna sesja nie działa. Klawiatura, mysz i nośniki wirtualne działają normalnie.',
        device: 'Urządzenie',
        busId: 'ID magistrali',
        speed: 'Prędkość',
        exporter: 'Eksporter',
        local: 'Zaimportowane jako',
        localValue: 'Magistrala {{bus}}, adres {{address}}',
        udc: 'Kontroler USB',
        pid: 'PID proxy',
        startedAt: 'Rozpoczęto',
        isoDevice:
          'To urządzenie nadaje przez izochroniczne punkty końcowe, czego na tym sprzęcie nigdy nie zmierzono',
        exporterLabel: 'Adres eksportera',
        exporterHint:
          'Host i port, z którymi łączy się NanoKVM. Przez tunel poniżej jest to {{exporter}}.',
        busIdLabel: 'ID magistrali na Twoim komputerze',
        busIdHint: 'Busid, który usbip list -l wypisuje dla urządzenia, na przykład {{example}}.',
        start: 'Uruchom przekazywanie',
        stop: 'Zatrzymaj przekazywanie',
        startTitle: 'Uruchomić przekazywanie USB?',
        startDevice: 'NanoKVM zaimportuje {{busId}} z {{exporter}}.',
        startHid:
          'Klawiatura USB, mysz i nośniki wirtualne przestają działać na czas trwania sesji i wracają samoczynnie po jej zatrzymaniu.',
        startIso:
          'Kamery internetowe i inne urządzenia izochroniczne wymagają włączenia przełącznika izochronicznego przed startem.',
        startWeb:
          'Ten interfejs webowy działa dalej, więc sesję można zatrzymać z tej strony w dowolnym momencie.',
        startNetwork:
          'Korzystaj z tej strony przez Ethernet lub Wi-Fi. Uruchomienie z sieci USB NanoKVM jest odrzucane, bo to połączenie by zniknęło.',
        okBtn: 'Uruchom',
        cancelBtn: 'Anuluj',
        instructions: 'Na Twoim komputerze',
        instructionsDesc:
          'Z założenia nie ma żadnego agenta do zainstalowania. Wykonaj te standardowe polecenia usbip na komputerze, do którego podłączone jest urządzenie.',
        copyFailed: 'Kopiowanie nie powiodło się. Skopiuj polecenie ręcznie.',
        copyInsecure: 'Ta strona nie jest serwowana przez HTTPS, więc przeglądarka zablokowała kopiowanie. Skopiuj polecenie ręcznie lub włącz HTTPS w Ustawieniach, Sieć.',
        directNote:
          'Bez tunelu usbipd musi być osiągalny w Twojej sieci, a adres eksportera powyżej musi go wskazywać. usbip przesyła urządzenie bez szyfrowania, więc lepszy jest tunel.',
        steps: {
          modprobe: {
            title: 'Załaduj sterownik po stronie eksportera',
            desc: 'usbip-host pozwala jądru oddać lokalne urządzenie. Domyślnie nie jest ładowany.'
          },
          list: {
            title: 'Znajdź urządzenie',
            desc: 'Wypisuje wszystkie lokalne urządzenia z busid oraz parą producent:produkt. Zapisz busid tego, które Cię interesuje.'
          },
          bind: {
            title: 'Podepnij je do usbip',
            desc: 'Odbiera urządzenie zwykłemu sterownikowi, więc przestaje ono działać na tym komputerze do czasu odpięcia.'
          },
          serve: {
            title: 'Udostępnij je',
            desc: 'usbipd pozostaje na pierwszym planie i czeka, aż NanoKVM zaimportuje urządzenie.',
            notice:
              'Standardowy usbipd nie ma opcji adresu nasłuchu i nasłuchuje na wszystkich interfejsach. Trzymaj port {{port}} zamknięty na zaporze i pozwól dotrzeć do niego tylko tunelowi poniżej.'
          },
          tunnel: {
            title: 'Skieruj go na NanoKVM',
            desc: 'Odwrotny tunel SSH: port {{port}} na pętli zwrotnej samego NanoKVM staje się usbipd na tym komputerze. Zostaw go uruchomionego na czas całej sesji.'
          },
          exporter: {
            title: 'Użyj tego jako eksportera',
            desc: 'Wpisz to w polu eksportera powyżej, podaj ID magistrali i uruchom sesję.'
          },
          unbind: {
            title: 'Oddaj urządzenie',
            desc: 'Po zatrzymaniu sesji to polecenie zwraca urządzenie zwykłemu sterownikowi na tym komputerze.'
          }
        }
      },
      mcp: {
        title: 'Usługa MCP',
        service: 'Zdalne sterowanie MCP',
        serviceDesc:
          'Zezwalaj zaufanym klientom MCP na sterowanie klawiaturą i myszą oraz wykonywanie zrzutów ekranu',
        securityWarning:
          'Każdy, kto ma ten klucz API, może sterować zdalnym hostem i wyświetlać jego ekran. Używaj HTTPS i włączaj usługę tylko w zaufanych sieciach.',
        endpoint: 'Punkt końcowy',
        apiKey: 'Klucz API',
        regenerateConfirmTitle: 'Wygenerować ponownie klucz API MCP?',
        regenerateConfirmDesc: 'Bieżący klucz natychmiast przestanie działać.',
        enableConfirmTitle: 'Włączyć zewnętrzne sterowanie MCP?',
        enableConfirmDesc:
          'Włączenie MCP zatrzyma PicoClaw i zamknie wszystkie aktywne sesje PicoClaw.',
        failed: 'Operacja MCP nie powiodła się',
        copyFailed: 'Kopiowanie nie powiodło się. Skopiuj ręcznie.',
        copyInsecure: 'Ta strona nie jest serwowana przez HTTPS, więc przeglądarka zablokowała kopiowanie. Skopiuj ręcznie lub włącz HTTPS w Ustawieniach, Sieć.',
        okBtn: 'Potwierdź',
        cancelBtn: 'Anuluj'
      },
      about: {
        title: 'NanoKVM - informacje',
        information: 'Informacje o systemie',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Wersja oprogramowania',
        applicationTip: 'Wersja aplikacji web NanoKVM',
        image: 'Wersja obrazu',
        imageTip: 'Wersja obrazu systemu NanoKVM',
        deviceKey: 'Klucz urządzenia',
        community: 'Społeczność',
        hostname: 'Nazwa hosta',
        hostnameUpdated: 'Zaktualizowano nazwę hosta. Uruchom ponownie, aby zastosować.',
        ipType: {
          Wired: 'Przewodowy',
          Wireless: 'Bezprzewodowe',
          Other: 'Inne'
        }
      },
      appearance: {
        title: 'Wygląd',
        display: 'Ekran',
        language: 'Język',
        languageDesc: 'Wybierz język interfejsu',
        webTitle: 'Tytuł strony internetowej',
        webTitleDesc: 'Dostosuj tytuł strony internetowej',
        menuBar: {
          title: 'Pasek menu',
          mode: 'Tryb wyświetlania',
          modeDesc: 'Wyświetl pasek menu na ekranie',
          modeOff: 'Wyłączone',
          modeAuto: 'Automatyczne ukrywanie',
          modeAlways: 'Zawsze widoczny',
          keyboardLedStatus: 'Wskaźniki blokad klawiatury',
          keyboardLedStatusDesc:
            'Wyświetl stan Num Lock, Caps Lock i Scroll Lock zdalnego komputera',
          icons: 'Ikony podmenu',
          iconsDesc: 'Wyświetla ikony podmenu na pasku menu'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Stan blokad zdalnej klawiatury',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'Włączone',
        off: 'Wyłączone',
        unknown: 'Nieznany'
      },
      device: {
        title: 'Urządzenie',
        oled: {
          title: 'OLED',
          description: 'OLED screen automatically sleep',
          0: 'Nigdy',
          15: '15 sec',
          30: '30 sec',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 godzina'
        },
        ssh: {
          description: 'Włącz SSH zdalny dostęp',
          tip: 'Ustaw silne hasło przed włączeniem (Konto - Zmień hasło)'
        },
        advanced: 'Ustawienia zaawansowane',
        swap: {
          title: 'Zamień',
          disable: 'Wyłącz',
          description: 'Ustaw rozmiar pliku wymiany',
          tip: 'Włączenie tej funkcji może skrócić żywotność karty SD!'
        },
        mouseJiggler: {
          title: 'Jiggler myszy',
          description: 'Uniemożliwia uśpienie zdalnego hosta',
          disable: 'Wyłącz',
          absolute: 'Tryb absolutny',
          relative: 'Tryb względny'
        },
        mdns: {
          description: 'Włącz usługę wykrywania mDNS',
          tip: 'Wyłączanie, jeśli nie jest potrzebne'
        },
        hdmi: {
          description: 'Włącz HDMI/wyjście monitora',
          idleTimeoutTitle: 'Limit czasu bezczynności przechwytywania',
          idleTimeoutDescription: 'Zatrzymaj przechwytywanie HDMI po czasie bez aktywnych widzów:',
          minutes: 'min'
        },
        autostart: {
          title: 'Ustawienia skryptów autostartu',
          description:
            'Zarządzaj skryptami uruchamianymi automatycznie podczas uruchamiania systemu',
          new: 'Nowy',
          deleteConfirm: 'Czy na pewno chcesz usunąć ten plik?',
          yes: 'Tak',
          no: 'Nie',
          scriptName: 'Nazwa skryptu autostartu',
          scriptContent: 'Treść skryptu autostartu',
          settings: 'Ustawienia'
        },
        hidOnly: 'HID – tylko tryb',
        hidOnlyDesc:
          'Przestań emulować urządzenia wirtualne, zachowując jedynie podstawową kontrolę HID',
        disk: 'Dysk wirtualny',
        diskDesc: 'Mount virtual U-disk on the remote host',
        network: 'Sieć wirtualna',
        networkDesc: 'Zamontuj wirtualną kartę sieciową na zdalnym hoście',
        networkProtocol: 'Protokół sieciowy',
        networkProtocolDesc: 'NCM dla nowoczesnych hostów, RNDIS dla starszych systemów Windows',
        rebindNotice: 'Przełączenie któregokolwiek z przełączników ponownie wylicza urządzenie USB, więc host docelowy na chwilę traci urządzenia wirtualne i sieć USB.',
        media: {
          title: 'Gniazda kamery i mikrofonu',
          desc: 'Zadeklaruj urządzenia multimedialne, które przeglądarki mogą zająć. Budżet punktów końcowych jest sprawdzany przy zastosowaniu profilu USB. Zapis powoduje ponowne wyliczenie urządzenia i rozłączenie podłączonych przeglądarek.',
          cameras: 'Kamery',
          microphones: 'Mikrofony',
          name: 'Nazwa',
          namePlaceholder: 'Widoczna na komputerze docelowym',
          addCamera: 'Dodaj kamerę',
          addMicrophone: 'Dodaj mikrofon',
          remove: 'Usuń',
          cameraDefault: 'Kamera NanoKVM {{index}}',
          microphoneDefault: 'Mikrofon NanoKVM {{index}}',
          nameRequired: 'Każde gniazdo wymaga nazwy.',
          budgetHint: 'Sześć punktów końcowych USB IN to sztywny limit sprzętowy. Umieść klawiaturę, mysz i wskaźnik bezwzględny na jednym interfejsie HID w sekcji Prezentacja USB albo wyłącz powyżej dysk wirtualny lub sieć USB.',
          unsupported:
            'To jądro nie potrafi nazwać urządzeń multimedialnych, więc komputery pokazują nazwę domyślną.',
          save: 'Zapisz gniazda',
          disconnect: 'Rozłącz',
          disconnectAll: 'Rozłącz wszystkie źródła',
          limit: 'Gniazda kamery i mikrofonu mogą łącznie wynosić najwyżej osiem.',
          failed: 'Nie udało się zaktualizować gniazd multimedialnych.'
        },
        reboot: 'Uruchom ponownie',
        rebootDesc: 'Czy na pewno chcesz ponownie uruchomić NanoKVM?',
        okBtn: 'Tak',
        cancelBtn: 'Nie'
      },
      network: {
        title: 'Sieć',
        wifi: {
          title: 'Wi-Fi',
          description: 'Skonfiguruj Wi-Fi',
          apMode: 'Tryb AP jest włączony, połącz z Wi-Fi skanując kod QR',
          connect: 'Połącz Wi-Fi',
          connectDesc1: 'Wprowadź SSID sieci i hasło',
          connectDesc2: 'Wprowadź hasło, aby połączyć się z tą siecią',
          disconnect: 'Czy na pewno chcesz rozłączyć sieć?',
          failed: 'Połączenie nie powiodło się, spróbuj ponownie.',
          ssid: 'Nazwa',
          password: 'Hasło',
          joinBtn: 'Połącz',
          confirmBtn: 'OK',
          cancelBtn: 'Anuluj'
        },
        tls: {
          description: 'Włącz protokół HTTPS',
          tip: 'Uwaga: użycie HTTPS może zwiększyć opóźnienie, szczególnie w trybie wideo MJPEG.'
        },
        bridge: {
          title: 'Mostek sieciowy',
          twoDevices:
            'Router widzi NanoKVM i sterowany komputer jako dwa osobne urządzenia, każde z własnym adresem.',
          loading: 'Ładowanie...',
          state: 'Stan',
          states: {
            disabled: 'Wyłączony',
            enabled: 'Włączony',
            rolledBack: 'Wycofano',
            failed: 'Niepowodzenie',
            pending: 'W toku'
          },
          uplink: 'Łącze nadrzędne',
          ports: 'Porty',
          protocol: 'Protokół urządzenia',
          up: 'aktywny',
          down: 'nieaktywny',
          noLink: 'brak łącza',
          enableTitle: 'Włączyć mostek sieciowy?',
          disableTitle: 'Wyłączyć mostek sieciowy?',
          reconnect:
            'Połączenie zarządzania zostanie na chwilę przerwane i nawiązane ponownie podczas przenoszenia adresu.',
          rollback:
            'Jeśli weryfikacja się nie powiedzie, poprzednia konfiguracja zostanie automatycznie przywrócona.',
          enableBtn: 'Włącz',
          disableBtn: 'Wyłącz',
          cancelBtn: 'Anuluj',
          interrupted:
            'Połączenie zostało przerwane podczas stosowania zmian. Trwa ponowne sprawdzanie stanu.',
          pendingNotice: 'Zmiana mostka wciąż trwa lub została przerwana przed zakończeniem.',
          revert: 'Przywróć poprzednią konfigurację',
          rolledBackNotice:
            'Ostatnia zmiana została wycofana, a poprzednia konfiguracja przywrócona.',
          verifyFailed: 'Weryfikacja nie powiodła się: {{gates}}',
          gates: {
            address: 'adres',
            gateway: 'brama',
            inbound: 'połączenie przychodzące'
          },
          inboundWeak:
            'Kontrola połączenia przychodzącego przeszła tylko dlatego, że NanoKVM połączył się sam ze sobą. Dowodzi to, że usługa sieciowa nasłuchuje i jest osiągalna lokalnie, a nie że żądanie z sieci do niej dociera.',
          noCarrier:
            'Brak łącza na porcie {{port}}. Mostek nie ma drogi do sieci, dopóki nie zostanie podłączony kabel.',
          loop: 'Router jest uczony także na porcie {{port}}, więc ten port jest drugą drogą do tej samej sieci. Spanning tree jest wyłączone, więc nic tutaj nie przerwie pętli: odłącz jedną z dwóch dróg.',
          failedNotice:
            'Nie udało się cofnąć ostatniej zmiany. NanoKVM może być dostępny tylko przez punkt dostępowy Wi-Fi lub konsolę szeregową.'
        },
        dns: {
          title: 'DNS',
          description: 'Skonfiguruj serwery DNS dla NanoKVM',
          mode: 'Tryb',
          dhcp: 'DHCP',
          manual: 'Ręcznie',
          add: 'Dodaj DNS',
          save: 'Zapisz',
          invalid: 'Wprowadź prawidłowy adres IP',
          noDhcp: 'Brak obecnie dostępnego DNS z DHCP',
          saved: 'Ustawienia DNS zapisane',
          saveFailed: 'Nie udało się zapisać ustawień DNS',
          unsaved: 'Niezapisane zmiany',
          maxServers: 'Dozwolone jest maksymalnie {{count}} serwerów DNS',
          dnsServers: 'Serwery DNS',
          dhcpServersDescription: 'Serwery DNS są automatycznie pobierane z DHCP',
          manualServersDescription: 'Serwery DNS można edytować ręcznie',
          networkDetails: 'Szczegóły sieci',
          interface: 'Interfejs',
          ipAddress: 'Adres IP',
          subnetMask: 'Maska podsieci',
          router: 'Router',
          none: 'Brak'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'Serwer VNC',
        description:
          'Pozwala dowolnemu klientowi VNC oglądać zdalny ekran oraz korzystać z klawiatury i myszy, po zalogowaniu kontem NanoKVM',
        port: 'Port',
        portDescription: 'Połącz się z tym portem pod adresem NanoKVM'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Optymalizacja pamięci',
          tip: "When memory usage exceeds the limit, garbage collection is performed more aggressively to attempt to free up memory. it's recommended to set to 50MB if using Tailscale. A Tailscale restart is required for the change to take effect."
        },
        swap: {
          title: 'Zamień pamięć',
          tip: 'Jeśli po włączeniu optymalizacji pamięci problemy nadal występują, spróbuj włączyć pamięć wymiany. Spowoduje to ustawienie domyślnego rozmiaru pliku wymiany na 256MB, który można dostosować w „Ustawienia > Urządzenie”.'
        },
        restart: 'Are you sure to restart Tailscale?',
        stop: 'Are you sure to stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable its automatic startup on boot.',
        loading: 'Ładowanie...',
        notInstall: 'Nie znaleziono Tailscale! Proszę zainstalować.',
        install: 'Instaluj',
        installing: 'Instalowanie',
        failed: 'Instalowanie nie powiodło się',
        retry: 'Odśwież stronę i spróbuj ponownie, albo spróbuj zainstalować manualnie.',
        download: 'Pobierz',
        package: 'pakiet instalacyjny',
        unzip: 'i wypakuj pliki',
        upTailscale: 'Prześlij tailscale do NanoKVM w katalogu /usr/bin/',
        upTailscaled: 'Prześlij tailscaled do NanoKVM w katalogu /usr/sbin/',
        refresh: 'Odśwież obecną stronę',
        notRunning: 'Tailscale nie działa. Rozpocznij, aby kontynuować.',
        run: 'Rozpocznij',
        notLogin:
          'Urządzenie nie zostało jeszcze powiązane. Zaloguj się i powiąż to urządzenie ze swoim kontem.',
        urlPeriod: 'Ten URL jest ważny przez 10 minut',
        login: 'Zaloguj',
        loginSuccess: 'Zalogowanie pomyślne',
        enable: 'Włącz Tailscale',
        deviceName: 'Nazwa urządzenia',
        deviceIP: 'Adres IP urządzenia',
        account: 'Konto',
        logout: 'Wyloguj',
        logoutDesc: 'Czy na pewno chcesz się wylogować?',
        uninstall: 'Odinstaluj Tailscale',
        uninstallDesc: 'Czy na pewno chcesz odinstalować Tailscale?',
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
        loading: 'Ładowanie...',
        notInstall: 'Nie zainstalowano',
        notConfigured: 'Nie skonfigurowano',
        stopped: 'Zatrzymano',
        running: 'Działa',
        connected: 'Połączono',
        error: 'Błąd',
        atBoot: 'uruchamia się przy starcie',
        notAtBoot: 'nie uruchamia się przy starcie',
        arguments: 'Argumenty',
        argumentsTip: 'Argumenty wiersza poleceń przekazywane usłudze przy uruchomieniu.',
        env: 'Zmienne środowiskowe',
        envKey: 'Nazwa',
        envValue: 'Wartość',
        envAdd: 'Dodaj zmienną',
        envRemove: 'Usuń',
        configured: 'Skonfigurowano',
        save: 'Zapisz',
        saved: 'Konfiguracja zapisana',
        start: 'Uruchom',
        stop: 'Zatrzymaj',
        restart: 'Uruchom ponownie',
        logs: 'Dzienniki',
        logsEmpty: 'Brak wpisów w dzienniku',
        refresh: 'Odśwież',
        binary: 'Plik binarny',
        binaryShipped: 'Dostarczony z firmware',
        binaryCustom: 'Własny plik binarny',
        binaryUpload: 'Prześlij plik binarny',
        binaryRevert: 'Przywróć plik z firmware',
        binaryRevertDesc:
          'Usunąć przesłany plik binarny i przywrócić wersję dostarczoną z firmware?',
        serverWarning: 'Serwer bez ograniczeń działa jak otwarte proxy',
        noHealthSignal:
          'Ta usługa nie zgłasza stanu, więc NanoKVM wie tylko, że proces działa, a nie czy tunel jest połączony.',
        memoryWarning: 'Uruchomienie kilku usług zdalnego dostępu naraz może wyczerpać pamięć',
        resources: 'Zasoby',
        memory: {
          title: 'Limit pamięci',
          description:
            'Ogranicza stertę Go usługi newt do {{limit}} MiB od jej najbliższego restartu. To jej własny limit, nie limit Tailscale; wyłączony pozostawia domyślną wartość Go, a GOGC=50 działa tak czy owak.',
          noRuntime:
            'wstunnel jest w Rust: nie ma odśmiecacza ani sterty do ograniczenia, a jego wątki robocze już podążają za jedynym CPU urządzenia.',
          notApplicable: 'Nie dotyczy'
        },
        swap: {
          title: 'Plik wymiany',
          description:
            'Dodaje plik wymiany 256 MB na karcie SD. Działa dla całego systemu: ta sama wymiana służy Tailscale, serwerowi KVM i wszystkiemu innemu na urządzeniu.'
        },
        okBtn: 'Tak',
        cancelBtn: 'Nie'
      },
      update: {
        title: 'Sprawdź aktualizacje',
        queryFailed: 'Uzyskanie wersji nie powiodło się',
        updateFailed: 'Aktualizacja nie powiodła się. Spróbuj ponownie.',
        isLatest: 'Oprogramowanie jest aktualne.',
        rebooting:
          'Trwa instalowanie nowego jądra i ponowne uruchamianie. Może to potrwać kilka minut; nie odłączaj zasilania.',
        kernelUpdate:
          'Ta aktualizacja instaluje jądro {{version}}. Urządzenie uruchomi się ponownie i samo wróci do bieżącego jądra, jeśli nowe nie wystartuje.',
        rolledBack:
          'Jądro {{version}} nie uruchomiło się, więc urządzenie wróciło do poprzedniego jądra.',
        available: 'Aktualizacja jest dostępna. Czy na pewno chcesz dokonać aktualizacji?',
        updating: 'Aktualizacja rozpoczęta. Proszę czekać...',
        confirm: 'Potwierdź',
        cancel: 'Anuluj',
        preview: 'Podgląd aktualizacji',
        previewDesc: 'Uzyskaj wcześniejszy dostęp do nowych funkcji i ulepszeń',
        previewTip:
          'Należy pamiętać, że wersje poglądowe mogą zawierać błędy lub niekompletną funkcjonalność!',
        customServer: {
          title: 'Niestandardowy serwer aktualizacji',
          desc: 'Sprawdzaj dostępność aktualizacji online i pobieraj je ze wskazanego serwera',
          invalidUrl:
            'Wprowadź prawidłowy adres katalogu serwera HTTP lub HTTPS, bez zapytania, fragmentu ani pliku latest.json.',
          loadFailed: 'Nie udało się wczytać konfiguracji serwera aktualizacji.',
          saveFailed: 'Nie udało się zapisać konfiguracji serwera aktualizacji.',
          saved: 'Konfiguracja serwera aktualizacji została zapisana.',
          save: 'Zapisz',
          confirmTitle: 'Użyć niestandardowego serwera aktualizacji?',
          confirmDesc:
            'SHA-512 sprawdza jedynie, czy pakiet jest zgodny z manifestem dostarczonym przez ten serwer. Nie potwierdza, że pakiet jest oficjalnym wydaniem NanoKVM. Wadliwy lub złośliwy serwer może unieruchomić urządzenie, spowodować utratę danych lub naruszyć bezpieczeństwo systemu.',
          confirm: 'Użyj mimo to',
          previewDisabled:
            'Aktualizacje w wersji testowej są niedostępne, gdy włączony jest niestandardowy serwer aktualizacji.'
        },
        offline: {
          kernelNotice: 'Ten pakiet zawiera jądro. Zostanie zapisane w zapasowym slocie, a urządzenie uruchomi się ponownie, aby je wypróbować; jeśli nie wstanie, samo wróci do bieżącego jądra.',
          kernelConfirm: 'Zainstaluj jądro',
          kernelCancel: 'Anuluj',
          title: 'Aktualizacje offline',
          desc: 'Aktualizacja poprzez lokalny pakiet instalacyjny',
          upload: 'Prześlij',
          checksumPlaceholder: 'Suma kontrolna SHA-256 (opcjonalnie)',
          invalidChecksum: 'Suma kontrolna SHA-256 musi zawierać 64 znaki szesnastkowe.',
          checksumMismatch: 'Weryfikacja SHA-256 nie powiodła się. Pakiet może być uszkodzony.',
          invalidName: 'Nieprawidłowy format nazwy pliku. Proszę pobrać z wydań GitHub.',
          updateFailed: 'Aktualizacja nie powiodła się. Spróbuj ponownie.'
        }
      },
      account: {
        title: 'Konto',
        webAccount: 'Nazwa konta web',
        role: 'Rola',
        roles: {
          admin: 'Administrator',
          user: 'Użytkownik'
        },
        password: 'Hasło',
        updateBtn: 'Update',
        logoutBtn: 'Wyloguj',
        logoutDesc: 'Czy na pewno chcesz się wylogować?',
        okBtn: 'Tak',
        cancelBtn: 'Nie',
        users: {
          title: 'Użytkownicy',
          create: 'Utwórz użytkownika',
          enabled: 'Włączony',
          disabled: 'Wyłączony',
          deviceOwner: 'Właściciel urządzenia',
          resetPassword: 'Zresetuj hasło',
          delete: 'Usuń',
          deleteConfirm: 'Usunąć tego użytkownika i unieważnić wszystkie jego sesje?',
          created: 'Użytkownik utworzony',
          deleted: 'Użytkownik usunięty',
          passwordUpdated: 'Hasło zaktualizowane',
          loadFailed: 'Nie udało się wczytać użytkowników',
          saveFailed: 'Nie udało się zapisać użytkownika',
          deleteFailed: 'Nie udało się usunąć użytkownika'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Asystent',
      empty: 'Otwórz panel i rozpocznij zadanie.',
      inputPlaceholder: 'Opisz, co chcesz, aby PicoClaw zrobił',
      newConversation: 'Nowa rozmowa',
      processing: 'Przetwarzanie...',
      agent: {
        defaultTitle: 'Asystent ogólny',
        defaultDescription: 'Ogólna pomoc dotycząca czatu, wyszukiwania i przestrzeni roboczej.',
        kvmTitle: 'Zdalne sterowanie',
        kvmDescription: 'Sterowanie zdalnym hostem poprzez NanoKVM.',
        switched: 'Rola agenta została zmieniona',
        switchFailed: 'Nie udało się zmienić roli agenta'
      },
      send: 'Wyślij',
      cancel: 'Anuluj',
      status: {
        connecting: 'Łączenie z bramką...',
        connected: 'Sesja PicoClaw połączona',
        disconnected: 'Sesja PicoClaw zamknięta',
        stopped: 'Wysłano żądanie zatrzymania',
        runtimeStarted: 'Runtime PicoClaw uruchomiony',
        runtimeStartFailed: 'Nie udało się uruchomić runtime PicoClaw',
        runtimeStopped: 'Runtime PicoClaw zatrzymany',
        runtimeStopFailed: 'Nie udało się zatrzymać runtime PicoClaw',
        controlSwitchedToMCP: 'Sterowanie przełączono na zewnętrzną usługę MCP'
      },
      connection: {
        runtime: {
          checking: 'Sprawdzam',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime gotowy',
          stopped: 'Runtime zatrzymany',
          blockedByMCP: 'Zewnętrzne sterowanie MCP jest aktywne',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime niedostępny',
          configError: 'Błąd konfiguracji'
        },
        transport: {
          connecting: 'Łączenie',
          connected: 'Połączono',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Bezczynność',
          busy: 'Zajęty'
        }
      },
      message: {
        toolAction: 'Akcja',
        observation: 'Obserwacja',
        screenshot: 'Zrzut ekranu'
      },
      overlay: {
        locked: 'PicoClaw steruje urządzeniem. Wprowadzanie ręczne zostało wstrzymane.'
      },
      control: {
        picoclaw: 'Sterowanie urządzeniem: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Sterowanie urządzeniem: zewnętrzny MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Sterowanie urządzeniem: wyłączone',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Przekaż sterowanie',
        release: 'Zwolnij',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'Sterowanie PicoClaw przyznane',
        released: 'Sterowanie PicoClaw zwolnione',
        grantFailed: 'Nie udało się przyznać sterowania PicoClaw',
        releaseFailed: 'Nie udało się zwolnić sterowania PicoClaw',
        grantConfirmTitle: 'Przełączyć sterowanie urządzeniem na PicoClaw?',
        grantConfirmDesc: 'Zapisy urządzenia przez zewnętrzny MCP zostaną przerwane.'
      },
      install: {
        install: 'Zainstaluj PicoClaw',
        installing: 'Instalowanie PicoClaw',
        success: 'PicoClaw zainstalowano pomyślnie',
        failed: 'Nie udało się zainstalować PicoClaw',
        uninstalling: 'Odinstalowywanie runtime...',
        uninstalled: 'Runtime został pomyślnie odinstalowany.',
        uninstallFailed: 'Odinstalowanie nie powiodło się.',
        requiredTitle: 'PicoClaw nie jest zainstalowany',
        requiredDescription: 'Zainstaluj PicoClaw przed uruchomieniem runtime PicoClaw.',
        progressDescription: 'PicoClaw jest pobierany i instalowany.',
        stages: {
          preparing: 'Przygotowanie',
          downloading: 'Pobieranie',
          extracting: 'Wypakowywanie',
          verifying: 'Weryfikowanie',
          installing: 'Instalowanie',
          installed: 'Zainstalowano',
          install_timeout: 'Upłynął limit czasu',
          install_failed: 'Niepowodzenie'
        }
      },
      model: {
        requiredTitle: 'Wymagana jest konfiguracja modelu',
        requiredDescription: 'Skonfiguruj model PicoClaw przed użyciem czatu PicoClaw.',
        docsTitle: 'Przewodnik konfiguracji',
        docsDesc: 'Obsługiwane modele i protokoły',
        menuLabel: 'Skonfiguruj model',
        modelIdentifier: 'Identyfikator modelu',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'Klucz API',
        apiKeyPlaceholder: 'Wprowadź klucz API modelu',
        save: 'Zapisz',
        saving: 'Zapisywanie',
        saved: 'Konfiguracja modelu została zapisana',
        saveFailed: 'Nie udało się zapisać konfiguracji modelu',
        invalid: 'Identyfikator modelu, API Base URL i klucz API są wymagane'
      },
      uninstall: {
        menuLabel: 'Odinstaluj',
        confirmTitle: 'Odinstaluj PicoClaw',
        confirmContent:
          'Czy na pewno chcesz odinstalować PicoClaw? Spowoduje to usunięcie pliku wykonywalnego i wszystkich plików konfiguracyjnych.',
        confirmOk: 'Odinstaluj',
        confirmCancel: 'Anuluj'
      },
      history: {
        title: 'Historia',
        loading: 'Ładowanie sesji...',
        emptyTitle: 'Nie ma jeszcze historii',
        emptyDescription: 'Tutaj pojawią się poprzednie sesje PicoClaw.',
        loadFailed: 'Nie udało się załadować historii sesji',
        deleteFailed: 'Nie udało się usunąć sesji',
        deleteConfirmTitle: 'Usuń sesję',
        deleteConfirmContent: 'Czy na pewno chcesz usunąć „{{title}}”?',
        deleteConfirmOk: 'Usuń',
        deleteConfirmCancel: 'Anuluj',
        messageCount_one: '{{count}} wiadomość',
        messageCount_other: '{{count}} wiadomości',
        messageCount: '{{count}} wiadomości'
      },
      config: {
        startRuntime: 'Uruchom PicoClaw',
        stopRuntime: 'Zatrzymaj PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Przełączyć sterowanie na PicoClaw?',
        enableConfirmDesc: 'Uruchomienie PicoClaw wyłączy zewnętrzną usługę MCP.',
        enableConfirmOk: 'Uruchom PicoClaw',
        enableConfirmCancel: 'Anuluj',
        title: 'Uruchom PicoClaw',
        description: 'Uruchom runtime, aby rozpocząć korzystanie z asystenta PicoClaw.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Wystąpił problem',
      refresh: 'Odśwież'
    },
    fullscreen: {
      toggle: 'Przełącz tryb pełnoekranowy'
    },
    menu: {
      collapse: 'Zwiń menu',
      expand: 'Rozwiń Menu'
    }
  }
};

export default pl;
