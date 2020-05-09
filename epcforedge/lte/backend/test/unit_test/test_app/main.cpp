/* SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2019 Intel Corporation
 */

//
//

#include "TesterFramework.h"

#include "PostUserplanesTester.h"
#include "PatchUserplanesTester.h"

#include "DelUserplanesTester.h"
#include "GetUserplanesTester.h"
extern int okNum ;
extern int ngNum ;

int main()
{
    // declarations here
    TesterFramework framework;

    // test cases


    printf("[==========] Starting register tester\n");
    PostUserplanesTester post_userplanes_tester;
    framework.registerTest(post_userplanes_tester, "PostUserplanes Test");

    PatchUserplanesTester patch_userplanes_tester;
    framework.registerTest(patch_userplanes_tester, "PatchUserplanes Test");
	

    GetUserplanesTester get_userplanes_tester;
    framework.registerTest(get_userplanes_tester, "GetUserplanes Test");


    DelUserplanesTester del_userplanes_tester;
    framework.registerTest(del_userplanes_tester, "DelUserplanes Test");

             
    printf("[==========] Running tester\n");

    // test execution starts here
    framework.fireAllTests();
    printf("\n[==========] Completed tester: OKNum=%d, TotalNum=%d\n", okNum, okNum + ngNum);

    return 0;
}
