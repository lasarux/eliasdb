/* 
 * EliasDB
 *
 * Copyright 2016 Matthias Ladkau. All rights reserved.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. 
 */

package paging

import (
	"testing"

	"devt.de/eliasdb/storage/file"
	"devt.de/eliasdb/storage/paging/view"
)

func TestFreePhysicalSlotManagerScale(t *testing.T) {
	sf, err := file.NewDefaultStorageFile(DBDIR+"/test5", false)
	if err != nil {
		t.Error(err.Error())
		return
	}

	psf, err := NewPagedStorageFile(sf)
	if err != nil {
		t.Error(err)
		return
	}

	if pc, err := CountPages(psf, view.TYPE_DATA_PAGE); pc != 0 || err != nil {
		t.Error("Unexpected page count result:", pc, err)
	}

	for i := 0; i < 5; i++ {
		_, err := psf.AllocatePage(view.TYPE_DATA_PAGE)
		if err != nil {
			t.Error(err)
		}
		if pc, err := CountPages(psf, view.TYPE_DATA_PAGE); pc != i+1 || err != nil {
			t.Error("Unexpected page count result:", pc, err)
		}
	}

	if pc, err := CountPages(psf, view.TYPE_DATA_PAGE); pc != 5 || err != nil {
		t.Error("Unexpected page count result:", pc, err)
	}

	r, err := sf.Get(1)
	if err != nil {
		t.Error(err)
		return
	}

	if pc, err := CountPages(psf, view.TYPE_DATA_PAGE); pc != -1 || err != file.ErrAlreadyInUse {
		t.Error("Unexpected page count result:", pc, err)
		return
	}

	sf.ReleaseInUse(r)

	r, err = sf.Get(3)
	if err != nil {
		t.Error(err)
		return
	}

	if pc, err := CountPages(psf, view.TYPE_DATA_PAGE); pc != -1 || err != file.ErrAlreadyInUse {
		t.Error("Unexpected page count result:", pc, err)
		return
	}

	sf.ReleaseInUse(r)

	if err := psf.Close(); err != nil {
		t.Error(err)
		return
	}
}
